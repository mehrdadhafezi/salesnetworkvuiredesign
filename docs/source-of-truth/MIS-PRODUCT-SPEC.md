# MIS PRODUCT SPEC

تاریخ: 2026-10-06 · **FROZEN PRODUCT TARGET / IMPLEMENTATION DEFERRED** · **PRODUCT / WHAT–WHY ONLY**.

## 1. Executive Summary

MIS مسئول **DATA OPERATIONS + SOURCE QUALITY + CUSTODY TRACEABILITY + GOVERNED REPORTING** است. وظیفه آن رساندن داده قابل ردیابی به فروش و آشکارکردن کیفیت/مغایرت است؛ broad data access آن را مالک همه تصمیم‌های فروش، مالی یا HR نمی‌کند. قواعد Gate 0 و اصل **KEEP + ADAPT** الزام‌آورند.

ساختار محصول برای طراحی Claude آماده است. Return safety، trust metrics، resolver coverage و maintenance authority شروط activation آینده‌اند؛ Freeze ادعای حل آنها یا موفقیت write نیست. هیچ کد، permission، workflow ذخیره‌شده یا business data تغییر نکرد؛ opening گزارش یک technical job خودکار ایجاد کرد و DB-pure بودن ادعا نمی‌شود.

مراجع:

- **A:** [Audit پذیرفته‌شده](CRM-PRODUCT-ROLE-ARCHITECTURE-AUDIT.fa.md)، بخش MIS و F01–F10.
- **G:** [Gate 0](GATE-0-PRODUCT-INVARIANTS.md)، INV-001…INV-036 و OPDهای باز.
- **C / S / D:** [Code Registry](CODE-EVIDENCE-AND-ACTION-REGISTRY.md)، [Status Catalog](STATUS-TRANSITIONS.fa.md)، [Decisions / Visibility](DECISIONS-VISIBILITY-INTERACTIONS.fa.md).
- مرزهای فروش: [Senior](SENIOR-SUPERVISOR-PRODUCT-SPEC.md)، [Manager](SALES-MANAGER-PRODUCT-SPEC.md)، [Deputy](SALES-DEPUTY-PRODUCT-SPEC.md)؛ Seller/Supervisor از A و تصمیم‌های G/D. Prototypeها منبع visual هستند، نه formula/permission/workflow.
- **L:** [MIS بازبینی 2026-10-06](../evidence/mis-reverification-2026-10-06.md)، با [inventory اولیه](../evidence/mis-loaded-panels-live.json) و [report evidence قبلی](../evidence/mis-report-details-live.txt) برای سابقه.

Evidence: **LIVE VERIFIED** همان مشاهده؛ **CODE VERIFIED** guard/query/transition در منبع پذیرفته‌شده؛ **BOTH** توافق همان claim؛ **INFERENCE** قرارداد target؛ **NOT VERIFIED** نبود شاهد کافی. Confidence موجودی و روابط کد HIGH؛ علت health mismatch، scale و successful write NOT VERIFIED. شماره نسخه فعال **2.0.123** یک بار دیده شد؛ byte equality با ZIP اثبات نشده. MIS #7 واقعی از HR وارد شد و پس از بررسی Admin بازگردانده شد.

## 2. Role Mission & Boundaries

| قرارداد | WHAT / WHY |
|---|---|
| Mission | ورود منبع معتبر، quality classification، traceable delivery/custody و reporting semantics governed |
| Daily Jobs | backlog source/batch؛ کیفیت ورود؛ assignment/delivery eligibility؛ بررسی ارتباط/مغایرت و report trust؛ actor بعدی |
| Occasional Jobs | import، plan/rule بازبینی‌شده، source lifecycle، export مجاز؛ maintenance فقط تحت اختیار مستقل |
| Decision Responsibilities | تعیین معتبر/invalid/duplicate با evidence؛ انتخاب source inventory واجد شرایط؛ delivery به target مجاز؛ تشخیص discrepancy و ارجاع domain owner |
| Data Needed | file/batch/run، native row ID و SourceRef/CaseRef، validity/reason، aliases/financial link evidence، current/history custody، timestamps، report metadata/results |
| Data Not Needed | HR credentials/compensation، commission rules، sensitive banking بدون نیاز، Finance posting tools، Sales actions خارج MIS authority، system settings |
| Allowed Responsibilities | source operations طبق guard، governed reads، diagnostics و repairs فقط با handler/policy اثبات‌شده؛ broad read≠repair-all |
| Non-Responsibilities | Sales Manager/Seller operator، Finance approve/refund/post/ledger، HR final/admin، system Admin، owner خودکار تمام business exceptions |

Mission **INFERENCE / HIGH**؛ daily/occasional شرح هدف‌اند، نه telemetry. MIS می‌تواند discrepancy را تشخیص دهد؛ تصمیم business outcome با domain مسئول می‌ماند. اصطلاح «پرونده» فعلی برای batch نباید با logical Case یک ردیف اشتباه شود.

## 3. Current Verified Inventory

۱۱ مقصد؛ وجود controls موفقیت write را ثابت نمی‌کند. Current snapshot تست کوچک و report امروز، proof large-batch نیست.

| Section | Evidence / current | Class | Decision / rationale |
|---|---|---|---|
| `mis-overview` / Today | valid1، waiting1، Sales1، returnable1؛ perbatch waiting0؛ L/C | ESSENTIAL | SIMPLIFY source/custody facts؛ overlap/financial ambiguity واضح، نه partition جعلی |
| `mis-import` | health warning و rebuild؛ L/C؛ import اجرا نشد | ESSENTIAL / CONFUSING health | KEEP Import؛ ADAPT readiness/freshness، no fake zero |
| Latest import result | quick-assignment summary به نام import، counters0؛ L | CONFUSING | RENAME operation context / SIMPLIFY؛ run type/source/time مشخص |
| `mis-batches-list` | assignment/return controls؛ A/C و L nav | ESSENTIAL | KEEP؛ return activation تحت F01، not trusted UI wording |
| `mis-pool` | source→pool/delivery؛ A/C و L nav | ESSENTIAL | KEEP؛ prepare pool≠live lead creation |
| `mis-preview` | quality/source/category/campaign breakdown؛ A/C | SUPPORTING | MERGE entry با Reports/Source Quality؛ raw quality view باقی |
| `mis-batches` | create/file/full delete/cleanup controls؛ L/C | SUPPORTING | MOVE lifecycle زیر Source Cases؛ batch/source grain واضح |
| Delete / cleanup controls | حساس؛ L/C | MISPLACED | MOVE Advanced/Maintenance؛ capability حذف/تغییر نشده |
| `mis-quick` | direct delivery پنج سطح؛ A/C | SUPPORTING | KEEP + CONDITIONAL؛ exceptional path، scope/reason policy جدا |
| `mis-distribution` | draft rule، plan preview/save؛ هیچ rule/plan؛ L/C | SUPPORTING | MOVE Planning؛ plan≠actual delivery |
| `mis-live` | approved plan، APPROVE/APPLY، dry-run/apply؛ L/C | SUPPORTING | MOVE conditional Planning/Advanced؛ materialization مستقل |
| `mis-logs` | diagnostic shell؛ A/L nav | SUPPORTING | KEEP در operations diagnostics؛ empty حذف نیست |
| `mis-report` | row1، historical Seller delivery1، invoice1، preinvoice1، completed0؛ L/C | ESSENTIAL | KEEP reconciliation contract؛ row≠global Case |
| XLSX same report | L control، C job payload | SUPPORTING | KEEP explicit export؛ اجرا/دانلود NOT VERIFIED |
| Reconciliation issue queue / trust context | کامل implemented بودن اثبات نشده | MISSING | ADD requirement با owner/proof؛ no unrestricted repair |

Target recommendations **INFERENCE**؛ presence/query agreements BOTH؛ raw controls وجود LIVE VERIFIED و mutation NOT VERIFIED. Code locators در [class-sn-plugin.php](../source/sales-network-v4/includes/class-sn-plugin.php): panel gate `11960`؛ import `21768`؛ assign `22008`؛ return `22053`؛ quick `22353`؛ pool `22605`؛ rule/plan `22674/22711`؛ approve/materialize `22786/22816`؛ delete/cleanup `19947/20069`.

## 4. MIS vs Sales / Finance

### MIS vs Sales Management

| Capability | Sales Management | MIS | Why Different? |
|---|---|---|---|
| Source ownership | source مصرف/visibility در scope | provenance/file/batch/quality | governing input با مالک sales جاری متفاوت |
| Customer/Case identity | operational context پرونده | trace native sources/aliases و ambiguity | neither phone merge authority؛ G resolver binding |
| Custody visibility | own/scoped work و team | source delivery/history و origin | current owner با assigned_manager source field یکی نیست |
| Direct assignment | own pool→eligible descendants | eligible source→Manager/pool یا exceptional پنج سطح | business scope/guards مستقل |
| Data repair | تشخیص/ارجاع، no raw repair | diagnostics و proven source repair conditional | broad view≠mutation عمومی |
| Import | مالک import MIS نیست | validated source import | ingestion نه فروش/تماس |
| Reports | تصمیم sales از facts | تعریف/کیفیت report contracts | report engineering اجرای فنی، نه تغییر schema توسط UI |
| Reconciliation | operational/invoice context | identify source/coverage/link/query discrepancy | outcome با domain owner |
| Financial links | Sales state/assist مجاز | dependency proof برای source safety | no financial approval/relink |
| Operational decisions | team workflow و next sales action | source eligibility/delivery/quality | MIS actor فروشنده یا Manager نمی‌شود |

### MIS vs Finance

| Capability | MIS | Finance | Boundary |
|---|---|---|---|
| Invoice visibility | scoped hard-link/state context | scoped financial review context | read≠approve |
| Payment data | proof مقدار/state برای dependency/report | validation/approved payment truth | MIS صفر یا paid جدید ایجاد نمی‌کند |
| Reconciliation | source/query/link mismatch | مالی/amount/ledger mismatch | مالک correction وابسته domain |
| Refund context | dependency/restriction و outcome مجاز | refund policy/proof/decision | cancel label اثبات bank refund نیست |
| Financial truth | مصرف read authoritative records | financial-domain decisions و ledger | report derived authority مستقل ندارد |
| Reporting | governed cross-source semantics | financial metrics/fields validated | مشترک query contract، action rights متفاوت |
| Source repair | proven nonfinancial source action conditional | invoice/payment facts مالک Finance | no automatic relink/reset |
| Posting | RESTRICTED | Finance-authorized process | outside MIS target |
| Approval | import/plan approvals مجاز در domain خود | financial review | APPROVE plan معادل approve invoice نیست |
| Ledger | فقط context ضروری مجاز | مالی/ledger authority | نه ledger admin یا commission rules MIS |

Finance رفتار role-specific در accepted Audit **CODE VERIFIED / NOT LIVE VERIFIED**؛ این جدول target boundaries است، نه success runtime. هیچ Finance audit جدید انجام نشده.

## 5. Target IA

| Module | Placement | WHAT / WHY / decision |
|---|---|---|
| Today / Work Queue | PRIMARY | next source action، health/trust و blocked items؛ KEEP + SIMPLIFY، truthful custody measures |
| Import & Data Quality | PRIMARY | file/run/parse/validity/duplicates و history؛ KEEP، readiness جدا از cached health |
| Source Cases | PRIMARY | native source rows و lineage در context batch/file؛ KEEP + ADAPT terminology |
| Custody / Delivery | PRIMARY | assign/prepare/deliver/return history؛ KEEP guards/domain separation |
| Reports / Reconciliation | PRIMARY | scoped governed dataset، discrepancy و resolution proof؛ KEEP reports + target issue context |
| Planning | SECONDARY / CONDITIONAL | rule/plan/approved materialization؛ MOVE مسیر occasional، no forecast/quotas |
| Advanced / Maintenance | ADVANCED / DIAGNOSTIC؛ writes CONDITIONAL | file/delete/cleanup/schema/reprocess/exceptional actions با boundaries مستقل؛ MOVE risky tools |
| Quick delivery | CONDITIONAL | technical پنج سطح موجود؛ exceptional bypass/reason policy؛ KEEP، no silent removal |

یک source picker/context مشترک می‌تواند entryهای preview/batch/report را مرتبط کند؛ query/rule آنها یکی نمی‌شود. Maintenance در daily workspace به‌عنوان ordinary next action ظاهر نشود؛ existing capabilities تا explicit rollout محفوظ. Target IA **INFERENCE / HIGH**؛ هیچ navigation visual طراحی نشده.

## 6. Import & Data Quality

File artifact، import batch، import run و stored row چهار مفهوم جدا؛ batch عنوان فعلی «پرونده» است ولی Case تجاری الزماً batch نیست. Target SourceRef طبق G شامل source domain/native ID، origin batch/actor/time و confidence است؛ encoding/storage schema آینده این Spec نیست.

| موضوع | Current evidence / target contract |
|---|---|
| Upload/parser | C `21768`: CSV/XLSX، 5MiB current file guard؛ valid batch/file و login/MIS nonce؛ no import executed |
| Native identity | stored `sn_mis_data_rows.id`، batch_id و row_number؛ row_number محل ورودی، global/canonical ID نیست |
| Provenance | file/batch/run، input row location، detected headers/phone fields، parser warnings و actor/time؛ lineage از فایل تا result بازیابی‌پذیر |
| Quality outcome | valid/duplicate/invalid با reasons؛ rows_seen/imported/skipped/failed جدا؛ parser failure≠empty valid upload |
| Duplicate classes | in-file، same batch reimport، other MIS row، legacy lead طبق C؛ phone-based duplicate finding proof canonical Case merge نیست |
| Retry/correction | stable run/source references و per-row known outcome؛ no blind reimport بعد Unknown؛ correction history با before/after، origin immutable |
| Unresolved linkage | evidence ambiguous→UNKNOWN؛ downstream financial/destructive write blocked؛ no fabricated alias |
| Import history | latest **import** run از latest operation جدا؛ type/batch/source/time/report counts؛ quick assignment report import نامیده نشود |

C import flow قبل parse برخی cleanup same-batch duplicate rows و schema repair را صدا می‌زند. **Import/preview را purely read یا no-cleanup تضمین نکنید**؛ current side effects CODE VERIFIED، execution NOT VERIFIED. Target ingestion و destructive maintenance باید product boundaries و truthful impact داشته باشند، بدون تغییر فعلی کد/permissions.

L readiness contradiction: import warning table missing/stale health با report data readable. Cause **NOT VERIFIED**؛ schema repair صحیح یا missing table قطعی اعلام نشود. Health state نیاز checked_at/source/error دارد و نباید operator را به repair blind سوق دهد. rejected/invalid sample محدود ممکن است full row outcome coverage نباشد؛ rejected خام حفظ/نمایش نیاز field/privacy policy، نه promise current storage.

No silent row merge؛ same phone می‌تواند چند Case معتبر داشته باشد. پذیرش یا skip duplicate فقط import policy domain است، نه overwrite customer/financial identity. Failed insert و partial parse باید در result باقی بمانند.

## 7. Case / Source Identity

G§2–3، INV-001…003 binding: **Case واحد کار تجاری با lineage معتبر**؛ Customer، Phone، Batch، Invoice و Event identities مستقل‌اند. native domain IDs با همان مقدار حفظ می‌شوند؛ equal ID در دو جدول یک Case نیست.

| مفهوم | قرارداد |
|---|---|
| Native IDs | MIS row، V4 distribution item، legacy lead، Dot case/invoice بومی؛ no ID rewrite |
| Canonical logical Case | immutable logical identity فقط از proof روابط معتبر؛ resolver format/DB انتخاب Engineering آینده |
| Aliases | native references همراه نوع رابطه و evidence؛ continuation versus related-but-distinct تحت OPD-01 |
| Unresolved aliases | keep originals، UNKNOWN confidence و incomplete coverage؛ no forced grouping |
| Source confidence | اثبات‌شده/ناقص/متناقض با proof source/time، نه score حدسی |
| Coverage | legacy+V4+MIS+Dot adapter coverage اعلام‌شده؛ MIS report row ID فقط grain محلی خود است |
| Unknown financial link | fail closed برای return/delete/reassign/reset وابسته؛ view تشخیصی با limit واضح |

phone فقط discovery attribute؛ نه join مالی قطعی، canonical ID یا مجوز merge. V4 invoice `lead_id=0` ممکن است با follow_invoice_id اثبات شود؛ legacy-only query آن را گم می‌کند. Dot creation خودکار اثبات Case جدید/همان Case نیست؛ alias proof لازم است. No forced DB merge، financial relink یا migration.

## 8. Custody / Delivery

Current `handle_mis_assign_batch:22008`: MIS/Admin gate + nonce، target sales_manager position، valid source rows با assigned_manager NULL؛ source manager/assigned_at و batch state/log تغییر می‌کند. **این assignment source با current custody distribution یکی نیست.** Active recipient/field/financial safety کامل از این guard محدود نتیجه نشود.

`handle_mis_prepare_lead_pool:22605`: candidate source rows→pool با source_row_id/batch/assigned_manager/status ready؛ apply با APPLY؛ limited candidate batch، report/log. `handle_mis_quick_assign_data:22353`: batch، target positions Deputy/Manager/Senior/Sup/Seller، limit حداکثر500 و APPLY؛ exceptional technical path، business bypass policy مشروط. هر دو source-level operation، نه unlimited recalls یا Seller sales action.

| Context | target meaning |
|---|---|
| Current Custody | current distribution owner/service state؛ از source assigned_manager به‌تنهایی حدس نزنید |
| Source Owner / provenance | actor/domain مسئول منبع؛ مالک فعلی sales نیست |
| Original Owner | origin ثابت طبق proof، نه hierarchy امروز |
| Next Actor | مسئول گام بعدی workflow؛ UNKNOWN اگر مسیر حل نشده |
| Credit Owner | attribution/entitlement تاریخی؛ receipt/custody current آن را عوض نمی‌کند |
| Event Actor / recipient | انجام‌دهنده و گیرنده واقعی transfer با timestamp/outcome |

L report historical Manager receipt0 و Seller delivery1 دارد، درحالی‌که current hierarchy همه Sales levels را دارد. این اختلاف خطا الزامی نیست؛ historical handoff با current subordinate relationship متفاوت است. Broad MIS report read در C دامنه داده منابع را می‌دهد، نه assignment بی‌حد در همه domains.

Target delivery: eligibility/source/state/custody و intended recipient در preview؛ apply recheck active/scope/current row/financial dependency؛ per-item results و audit؛ transfer event/relationships محفوظ. آماده pool، تحویل distribution و controlled live lead materialization workflows جدا و KEEP هستند. No generic ACK/notification/SLA یا automatic credit transfer.

## 9. Return Safety

**F01 P0:** L یک row دارای hard-linked preinvoice و overview returnable1 نشان داد؛ return اجرا نشد. C V4 issuance lead_id0 + follow_invoice_id را می‌نویسد، ولی `sn_distribution_item_has_invoice_lock:22033` فقط virtual lead_id و noncancelled/nonrejected invoices را می‌بیند. MIS fallback/apply `22053/22296` می‌تواند owner/contact/customer state را reset کند بدون full financial-link recheck. Batch deletion preflight برعکس follow link دارد؛ protection بین عملیات یکسان نیست. **CODE VERIFIED gap؛ unsafe data loss runtime NOT VERIFIED**.

Source return، unassign، recall و هر ordinary destructive reset تحت G§6، INV-006…010:

1. authority از source/action scope و actual custody/transfer proof؛ مقام MIS یا broad data visibility recall-all نیست.
2. valid financial relationships union: legacy lead_id، V4 follow_invoice_id، Seller flow state/event last invoice، Dot financial relations و سایر protected fulfillment proof.
3. active valid dependency consumption را block کند؛ unknown/missing/contradictory links **fail closed**؛ نبود invoice در یک view proof عدم مصرف نیست.
4. cancelled/rejected no auto-release؛ OPD-03 reset/release policy unresolved؛ refund_confirmed label بانک proof نیست.
5. preview≠apply؛ scope/state/custody/all links دوباره با race protection، no blind reset/retry.
6. source/native aliases، invoice relationships، transfer/contact/financial history و credit attribution دست‌نخورده؛ truthful per-item blocked/applied/error/unknown.

Requirement Freeze است؛ **fix اجرا نشده**. current UI «هنوز فاکتور نشده/قابل برگشت» trusted permission proof نیست. source display و return predicate باید قبل activation reconcile شوند؛ طراحی read diagnosis ممکن، unsafe reset آینده نیاز QA/rollback.

## 10. Data Quality / Reconciliation

Reconciliation workspace target **INFERENCE**؛ source/guards از A/C. تشخیص مسئله authority اصلاح همه domains نمی‌دهد. repairs جدول زیر مجوز موجود جدید ایجاد نمی‌کنند؛ فقط action proved با policy/impact guard آینده قابل فعال‌سازی است.

| Class / owner | MIS responsibility / repair boundary | Another domain / resolution proof |
|---|---|---|
| Identity mismatch؛ source/resolver owner | inspect native refs/provenance؛ proven nonfinancial correction conditional | invoice/customer identity relink به actor مجاز؛ explicit lineage proof، no phone merge |
| Source coverage mismatch؛ report/source owner | identify missing adapter/population و mark incomplete | report engineering/source owner؛ same-cohort coverage اعلام‌شده |
| Duplicate source row؛ MIS source owner | duplicate trace/reimport class؛ repair فقط proved safe cleanup | protected sales/financial links blocked؛ retained original refs/audit و remaining counts |
| Unresolved alias؛ resolver owner | preserve inputs و UNKNOWN؛ no merge | domain owner اثبات continuation؛ auditable relation evidence |
| Missing custody؛ delivery owner | compare source assignment/pool/item/history | current Sales responsibility با sales owner؛ valid transfer chain، no arbitrary reassignment |
| Impossible hierarchy؛ HR owner | report contradiction/read scope، no HR edit | HR validates current/effective history؛ original attribution محفوظ |
| Invoice/report mismatch؛ report/Finance owners | align grain/time/cohort/source و identify links | Finance changes money/review؛ engineer fixes query؛ exact parity explanation |
| Stale report؛ report owner | freshness/expiry، permitted read rebuild | technical report service؛ new gather window/coverage، نه fake as-of |
| Partial import؛ MIS source owner | per-row outcomes و safe known-failure retry | parser/technical owner؛ count reconciliation، no double import |
| Unknown write outcome؛ action owner | reread logs/result IDs قبل retry | responsible action service؛ definitive outcome/correlation proof |
| Orphan-looking relation؛ source/domain owner | inspect hard links and history؛ orphan-looking≠unused | financial/Sales owner validates protected relation؛ no delete because UI missing |
| Health contradiction؛ technical owner | display checked_at/source و distinguish warning | schema/diagnostics owner؛ verified readiness without misleading last-op counts |

Diagnostic-only actions READ؛ correction queue ownership/routing تازه **DEFERRED**. Format-valid reason یا `recorded` label صحت معنایی عدم خرید را ثابت نمی‌کند. Repair history/source changes نباید historical sales/finance truth را overwrite کنند.

## 11. Reports & KPI Trust

MIS governs semantics/coverage و traceability؛ business completion/payment تصمیم Finance/Sales domain است. [MIS report model](../source/sales-network-v4/includes/reports/class-sn-mis-report-model.php)، [source](../source/sales-network-v4/includes/reports/class-sn-mis-report-source.php) و [store](../source/sales-network-v4/includes/reports/class-sn-mis-report-store.php) sources C هستند. Report source برای MIS/Admin در dataset خود unbounded descendant constraint دارد؛ Sales readers descendants scoped؛ این broad read اختیار Admin business actions نیست.

### Query contract

Required: query/filter contract و **metric version**، source coverage، grain، scope، cohort، time_basis/timezone، predicate/formula، freshness/gather window، bounded/partial indicator و export parity. version یک product metadata requirement است، نه تغییر DB schema. KPI فاقد این‌ها **NOT FINAL**.

Current report: unique source case = `sn_mis_data_rows.id`، distinct invoice IDs و delivery events/persons جدا؛ hard link از follow/state/events، نه phone-only. selected source cases می‌توانند همه invoices linked خود را در sum داشته باشند؛ این خودکار invoice-issued-in-range نیست. invoice مشترک در global aggregation یک‌بار؛ level/row attribution amounts جمع‌شدنی بی‌قید نیستند.

| Time semantics | معنی |
|---|---|
| Current Snapshot | current state هنگام read/gather؛ scope/refresh time معلوم |
| Event Range + Gathered Current State | پرونده با رویداد در بازه انتخاب، سپس وضعیت جاری در gather؛ event cohort و measurement time جدا |
| Historical As-Of | reconstructed historical state در لحظه گذشته فقط با evidence کافی؛ current gathered fields proof نیست |

L اکنون F09 توضیح را صریح نشان می‌دهد: statuses جاری هنگام گردآوری؛ gather start/end و +03:30. **F09 حل historical-as-of نشده**؛ wording روشن current implementation دلیل ادعای historical reconstruction نیست. stepped gather snapshot یک لحظه atomic DB نیست؛ source changes حین steps limits explicit.

### Metrics / findings

| Finding | Product meaning / trust | Reconciliation/display contract | Correction owner |
|---|---|---|---|
| F02 | `pre_invoice` در برخی Sales predicates جا افتاده؛ confirmed C و Deputy L قبلی | Open Pre-invoices snapshot، Pre-invoices Issued history و Invoice Created distinct؛ do not silently align0→1 | report/Engineering برای predicate؛ business dictionary مالک domain، G |
| F03 | legacy-only counts Total Cases نیست | label Legacy Leads Only یا governed source union/resolver؛ missing sources incomplete | source/report owner؛ canonical mapping OPD-01 |
| F04 | Manager report0 vs invoices1 accepted/fresh؛ MIS sample invoice1 خود fix نیست | same scope/cohort/time/grain/source؛ discrepancy flag و explanation، no fabricated zero | query/report owners؛ Finance فقط financial fact corrections |
| F09 | event-selection همراه current measurement، نه As-Of | دو زمان explicit؛ bounded/gather freshness؛ Historical As-Of conditional | report semantic/engineering owner، no history rewrite |

Target metric predicates: **Invoice Created** distinct creation cohort؛ **Open Pre-invoices** valid current preinvoice dictionary؛ **Pre-invoices Issued** issuance history؛ **Sales Completed** approved full-value collection با G tolerance/amount conservation؛ **Total Cases** logical Case union/proved aliases؛ **Legacy Leads Only** sn_leads-only. هر یک grain/source/cohort/time/scope/freshness دارد؛ OPD-06/07 rounding/unit/cohort/attribution باز، نرخ/target/ranking اختراع نشود. approved stage≠full invoice completion، financial history state≠contact followup.

Current Today waiting1/withSales1/returnable1 و perbatch waiting0 independent predicates‌اند؛ partition sum صحیح نیست. Label/unit گزارش ذخیره‌شده و actual unit metadata نیازمند reconciliation؛ 15,000 را بدون conversion proof تومان/ریال دوباره تفسیر نکنید.

### Export

Explicit export permission مستقل از read؛ same authorized dataset/scope و field whitelist؛ sensitive bank/HR/finance fields restricted؛ source/time/cohort/metric-version/gather/bounded metadata همراه output. XLSX current payload همان report job در C؛ مالک user، scope hash، expiry و access recheck وجود دارد. job limit3 active و expiry1hour C، current runtime negative checks NOT VERIFIED. CSV یا هر format نباید row/field policy را bypass کند. Export/download اجرا نشده؛ read report technical cache/storage است نه business write یا DB-pure guarantee.

## 12. Planning

Actual L/C: rule draft (equal/manual)، manager selector، batch/source و selected Seller candidates؛ plan preview/save، plan approval و controlled live materialization. No rule/plan sample؛ متن «فروشنده‌ای در scope نیست» کنار Manager موجود؛ candidate completeness **NEEDS VALIDATION**، hierarchy repair یا absence Seller قطعی نیست.

| Stage | Current supported semantics / target boundary |
|---|---|
| Rule draft | plan instruction، نه immediate assignment یا forecast |
| Plan preview/save | source/batch/Manager/rule/Seller input؛ save plan≠deliver/live Seller lead |
| Approve | C `22786`: draft/dry_run + APPROVE؛ reason/log؛ report approved flag code does not prove DB update success under failures |
| Materialize | C `22816`: approved eligibility report، APPLY؛ create live lead path و counts؛ duplicate/existing leads checks source-dependent، full race behavior NOT VERIFIED |
| Capacity input | selected actors/eligible inventory؛ formal capacity formula، import schedule/forecast/quotas اثبات نشده |

Controlled materialization طبق UI special path، no SMS/update/delete existing leads؛ UI text proof all-runtime safety نیست. dry-run handler ممکن است report/options/schema/cleanup side effects داشته باشد؛ **preview ساده را purely read تضمین نکنید و این مرحله اجرا نکردید**. No blanket atomic all-success claim؛ per-item outcomes حتی در failure after eligibility ضروری.

Target Planning SECONDARY/CONDITIONAL، construction/rule→plan→approval→materialization جدا. skip-level business policy، maker/checker OPD-08 و full hierarchy planning **DEFERRED**. هیچ forecasting، quota، automated import schedule یا sales capacity score ایجاد نشود.

## 13. Advanced / Maintenance

Classification درباره placement/authority target است؛ current gates اغلب MIS/Admin هستند، «Advanced» admin-only proof نیست. هیچ قابلیت موجود حذف/تغییر نشده.

| Action | Classification | Target rationale / current evidence |
|---|---|---|
| View source/import/log/report | NORMAL OPERATION / diagnostic READ | daily provenance/trust؛ field/security controls |
| Ordinary source assign/pool delivery | NORMAL OPERATION | source eligibility guards؛ F01 وابسته برای destructive effects |
| Quick five-level delivery | ADVANCED / CONDITIONAL | exceptional technical path، bypass reason/policy open |
| Draft rule/plan | ADVANCED planning | occasional، no immediate business delivery |
| Controlled live-lead creation | ADVANCED / CONDITIONAL | approved plan/confirm/duplicate guards و QA |
| Delete source file | MAINTENANCE | source artifact≠records/provenance؛ current guard DELETE_FILE + preflight؛ actual impact not executed |
| Delete full batch | MAINTENANCE / RESTRICTED while consumed/unknown | DELETE_BATCH و locked-items preflight C `19947`؛ target all-link/history safety |
| Cleanup duplicate/deleted traces | MAINTENANCE ONLY | current `20069/21247` paths؛ lock/lineage/financial impact validated، no phone-only orphan deletion |
| Schema safe rebuild/repair | MAINTENANCE / RESTRICTED technical | health-warning control L؛ no successful/safe reconstruction claim؛ system dependency owner |
| Repair linkage / financial relink | NEEDS VALIDATION / RESTRICTED | no general MIS repair entitlement؛ G prohibits automatic financial relink |
| Reprocess / rebuild derived data | NEEDS VALIDATION | read report rebuild differs from repairing source or replaying business side effects |
| Read-only reconcile | NORMAL diagnostic | identify discrepancies/owner؛ doesn't modify ledger |
| Exceptional reassignment | ADVANCED / NEEDS POLICY | source/current custody/financial guards، not unbounded recall |

Maintenance impact preview must identify artifact/row/link/history affected، protected/unknown blocks، actor/reason/result و rollback requirements. Current delete preflight proof target كامل financial-safe cleanup نیست؛ retention/rollback policy باز. Maintenance daily workspace را dominate نکند، disabled current permission فرض نشود. **FUTURE IMPLEMENTATION — REQUIRES QA / ROLLBACK**.

## 14. Permissions / Bulk Actions

Conceptual product permissions زیر WP capability جدید نیستند. common read layers: login/role entry، module authority، row/source scope و field restriction؛ C `sn_can_access_mis_panel:11960` Admin یا authenticated MIS position، نه per-action separation کامل. Broad entry همه sensitive repairهای target را مجاز نمی‌کند.

| Concept | Assessment |
|---|---|
| `view_mis_today` | KEEP؛ trustworthy predicates/coverage ADAPT |
| `view_import_batches` | KEEP source/file/run context |
| `import_source_data` | KEEP supported parser؛ cleanup/read purity boundary و truthful row outcomes |
| `view_source_cases` | KEEP native rows/source context؛ logical Case coverage conditional |
| `view_case_lineage` | target traceability؛ partial resolver proof، UNKNOWN حفظ |
| `view_custody_history` | KEEP event/custody contexts مستقل |
| `assign_source_cases` | KEEP eligible source assignment/delivery؛ no unlimited sales operation |
| `return_source_cases` | CONDITIONAL guarded rollout، F01 critical |
| `view_data_quality` | KEEP valid/duplicate/invalid و source health |
| `reconcile_data` | READ diagnosis supported؛ write correction separate conditional policy |
| `view_reports` | KEEP source/scope/time/grain governed |
| `export_reports` | CONDITIONAL explicit scope/field/security parity؛ runtime NOT VERIFIED |
| `run_maintenance_action` | MAINTENANCE ONLY، action-specific impact/guard؛ blanket entitlement نیست |

Write/approval guards target؛ موجودبودن کل requirements در کد ادعا نمی‌شود:

| Action | Scope / state guard | Row guard / field restriction | Audit requirement |
|---|---|---|---|
| Import | allowed source/batch/parser؛ readiness و policy، no arbitrary domain write | input row identity/known retry state؛ no overwrite canonical/financial identity | run/file/batch، row outcome/reason، actor/time/correlation |
| Assign batch | permitted source inventory/target role-active policy؛ valid unassigned | current source assignment/custody/consumption recheck؛ only allowed destination fields | before/after refs، affected/skipped counts |
| Prepare/deliver/quick | eligible source/recipient، permitted path، current status | source/pool/item/native link proof؛ no credit/history reset؛ active/scope recheck | source→recipient event، reason/results |
| Return/unassign/recall | own permitted return basis + all financial links nonconsumed | custody/transfer/identity/race verified؛ protected history/links immutable | per-row blockers/outcomes/history، G§6 |
| Plan approve/materialize | permitted plan/source/actor scope؛ draft/dryrun→approved、approved→eligible apply | plan/item unchanged/duplicate guards؛ no update existing lead without authority | plan/run/reason/decision/materialized refs، partial results |
| Cleanup/delete/repair | independent maintenance authority؛ protected/unknown fail closed | full impact links/row guards؛ no automatic financial relink/ID rewrite | before/after impact، reason/time/result، rollback proof |
| Export | job owner/current scope and permission、expiry/contract valid | authorized row/field whitelist؛ sensitive fields excluded | job/query/source/time/correlation/coverage |

| Bulk | Classification | Rationale |
|---|---|---|
| Bulk Import | REQUIRED / SUPPORTED code | core ingestion؛ runtime/scale NOT VERIFIED، per-row parsed/inserted/rejected/skipped/failed/unknown |
| Bulk Assignment | REQUIRED / SUPPORTED controls | source batch/pool delivery؛ actual row guards apply، limited candidate scope واضح |
| Bulk Return | SUPPORTED code / CONDITIONAL activation | current path exists؛ F01 full protection/QA prerequisite |
| Bulk Reconcile | CONDITIONAL read / NEEDS VALIDATION write | reporting comparison versus business correction مستقل |
| Bulk Repair | MAINTENANCE ONLY / NEEDS VALIDATION | scope/impact/policy per action؛ repair-all اختراع نشود |
| Bulk Export | SUPPORTED code / CONDITIONAL permission | same dataset/scope/fields、job bounds/expiry؛ execution NOT VERIFIED |
| Bulk Delete/Cleanup | MAINTENANCE ONLY؛ NOT ALLOWED protected/unknown | existing controls حفظ؛ target safety gates و rollback |

Truthful per-item outcomes INV-034: requested/eligible/applied/skipped/rejected/failed/unknown، counts reconcile؛ aggregate fake all-success ممنوع. batch count/summary alone proof every row success نیست؛ known failed retry با fresh guards، Outcome Unknown ابتدا reconcile.

## 15. Auditability

Required product context: **actor، action، source، batch/run، CaseRef/SourceRef/native refs، before/after where appropriate، reason، timestamp/timezone، result، correlation ID و per-item partial outcome**. IDs باید عملیات را به records موجود وصل کنند؛ correlation logical requirement است، نه schema جدید اجباری.

| Operation | Audit proof needed |
|---|---|
| Import | file/run/source rows، parser/quality reasons، inserts/skips/errors؛ latest operation از import history جدا |
| Delivery/Return | from/to/current row، actor event و timestamp؛ source/financial blockers؛ preservation of previous history |
| Plan/materialize | rule/plan/state/approval/run، resulting lead refs، failures؛ plan approval≠all leads created |
| Reconcile/correction | mismatch observed، domain owner، query metadata؛ approved correction قبل/بعد با untouched protected refs |
| Maintenance | impacted artifacts/rows/links و blocked items، reason/authority، outcome/rollback context |
| Report/export | job/query/version/scope/time/coverage/gather window، owner/access expiry؛ technical cache with no business mutation |

Current sn_mis_log و report options/history وجود دارند؛ exhaustive immutable audit، correlation و transactional completeness **NOT VERIFIED**. overwritten last-report option تاریخچه کامل نیست. before/after customer/HR/bank data فقط field-policy ضروری؛ credentials در audit نباید expose شوند. Retention policy/technical schema deferred، no history rewrite یا deletion برای ساده‌سازی.

## 16. Data Visibility / States

| Visibility | Contract |
|---|---|
| MUST SEE | source/file/batch/run و native refs، validity/reason/duplicate evidence، linkage confidence، custody/history/actor/next actor، financial dependency block، report grain/cohort/scope/time/source coverage، action outcomes |
| NICE TO HAVE | filters/source segmentation، diagnostic link proof و history context، allowed parser warnings/samples، expiry/build progress؛ no unapproved score |
| SHOULD HIDE / RESTRICT | Finance approval/posting/refund tools، HR credentials/compensation، commission rules، out-of-authority Sales actions، unnecessary bank data، system-admin controls |

Broad source access به معنی broad action authority نیست. MIS technical diagnostics نیاز namespace/source IDs دارند، اما personal/sensitive fields minimum necessary؛ هر read/export whitelist مستقل.

| State | MIS example / meaning |
|---|---|
| Loading | import parse/report gather pending؛ final total premature نیست |
| Empty | authorized complete source scope without rows؛ readiness failure نیست |
| No Result | valid filter no match؛ whole source empty فرض نشود |
| Unauthorized | source/module/row/field/action denied؛ mask with zero ممنوع |
| Stale | health/query/cache قدیمی؛ checked_at/gather time مشخص |
| Incomplete | partial parsed import، missing source/alias، bounded report؛ totals NOT FINAL |
| Conflict | duplicate/unresolved identity، lineage/custody/state changed؛ no silent merge/reset |
| Partial Success | rows inserted/delivered و rows rejected/skipped/failed مستقل |
| Retryable Failure | known parser/read/insert failure with safe eligibility recheck |
| Outcome Unknown | write result نامعلوم؛ audit/current rows reconcile before reimport/return |
| Offline | connectivity lost؛ cached report stale، pending write completed فرض نشود |

Row rejected/duplicate quality state با transport failure یکی نیست. Report bounded یعنی full population نشده، نه failed؛ reconcile mismatch diagnosis است، نه اجازه write. Maintenance blocked یک outcome صحیح حفاظت است؛ unknown financial relation «unused/orphan» نامیده نشود. Health inconsistent و source incomplete هرگز legitimate zero نسازند.

## 17. Product Gaps / Deferred

| ID / Priority | Status / Evidence / confidence | شرط / limit |
|---|---|---|
| MIS-G01 / P0 — F01 return/reset | PRODUCT GAP — DO NOT IMPLEMENT؛ A/C/L HIGH | all valid links+race/failclosed/history پیش از future activation؛ runtime loss NOT VERIFIED |
| MIS-G02 / P1 — resolver coverage | DEFERRED / CONDITIONAL؛ G/OPD-01 | cross-domain aliases/proof و logical Case union؛ row ID یا phone global canonical نشود |
| MIS-G03 / P1 — health/readiness | PRODUCT GAP — DO NOT IMPLEMENT؛ L HIGH contradiction، cause UNKNOWN | fresh schema/health proof؛ no blind repair یا fake import0 |
| MIS-G04 / P1 — F02/F03/F04 definitions | PRODUCT GAP — DO NOT IMPLEMENT؛ A/C و fresh sales specs | source/grain/cohort parity؛ UI redesign correction proof نیست |
| MIS-G05 / P1 — F09/as-of | DEFERRED / CONDITIONAL؛ C/L HIGH semantics | event cohort+current gather واضح؛ true historical reconstruction proof لازم |
| MIS-G06 / P2 — bounded/stepped reports | PRODUCT GAP — DO NOT IMPLEMENT؛ C/A | candidate/full coverage، gather window و concurrency limits؛ atomic snapshot ادعا نشود |
| MIS-G07 / P1 — repair/maintenance authority | DEFERRED / CONDITIONAL؛ C guards | independent policy/field/row impact؛ diagnostics≠unrestricted mutation |
| MIS-G08 / P2 — planning | NEEDS VALIDATION؛ L/C controls، no plan dataset | actual Seller candidates، hierarchy layers، approval outcome، OPD-08 maker/checker؛ no forecast/quota |
| MIS-G09 / P1 — large-batch outcomes/retry | NEEDS VALIDATION؛ C loop/bounds | truthful row outcomes، interruption/idempotency/partial failure QA؛ no all-success |
| MIS-G10 / P1 — export parity/security | DEFERRED / CONDITIONAL؛ C job reuse، L presence | fields/scope/cohort/time/expiry negative tests؛ download not performed |
| MIS-G11 / P2 — latest operation/history | PRODUCT GAP — DO NOT IMPLEMENT؛ L | import/result run type و history از overwrite summary جدا |
| MIS-G12 / P2 — reconciliation owner | DEFERRED؛ G/OPD-02 | source/report/Finance/HR/Sales responsible actors؛ new escalation command/SLA نه |

OPD-03 cancelled-release، OPD-06/07 currency/cohort/attribution، OPD-09 entitlement و OPD-10 Sales delegation بازند؛ MIS UI آنها را با schema patch یا mock business rule حل نکند. P3 optimization/scheduling تنها پس از needs/scale evidence. نبود full historical-as-of یا repair authority مانع طراحی diagnosis/read workspace نیست؛ activation features conditional.

## 18. Compatibility / Safe Adoption

تمام recommendations **KEEP + ADAPT**. Source IDs، Case/native IDs، aliases، histories، transfer events، invoice relationships، legacy data، routes/status namespaces، current workflows و permissions تا rollout صریح حفظ شوند.

| Adaptation | Compatibility bridge / future acceptance |
|---|---|
| Today/Source terminology | batch/file/row/Case context honest؛ existing IDs/routes unchanged، no silent total formula |
| Import quality/history | current parser/duplicate semantics traced؛ no automatic merge/overwrite؛ cleanup separation QA |
| Custody/Delivery | source assignment/pool/live lead distinction؛ source/current/history identity محفوظ |
| Return guards | all-source financial protection و race checks؛ no protected resets/relink |
| Reports | governed adapters/metadata، row-grain explicit، export parity؛ event/current/as-of distinct |
| Planning | current technical exceptional path محفوظ؛ conditional policy، no unintended lead updates |
| Maintenance placement | presentation move و action-specific guards future؛ current capability silently revoke نشود |

No destructive merge، ID/history rewrite، forced migration، status rename، automatic financial relink یا attribution backfill حدسی. **FUTURE IMPLEMENTATION — REQUIRES QA / ROLLBACK** برای risky changes؛ rollback مالی/history را پاک نکند.

Future QA: multi-format import partial/duplicates/reimport interruption؛ financial-protected cleanup؛ all-source return links/race/cancelled ambiguity؛ delivery state/custody/recipient changes؛ plan approval failure/per-item materialize؛ same-cohort metrics و shared invoice dedup؛ stepped/bounded report freshness؛ scope/user/expiry/field export parity؛ read versus maintenance side effects؛ unresolved aliases preserved. هیچ write QA/implementation در این مرحله اجرا نشده.

## 19. Claude Design Handoff

### WHAT CLAUDE MAY ASSUME

MIS Mission و Target IA بخش5 Freeze‌اند؛ daily source/quality/custody/reporting، occasional Planning و separated Maintenance. Gate0 contracts، existing IDs/history و role boundaries binding؛ source/report states مستقل. current technical paths حفظ، conditional authority روشن. Sales prototypes visual foundations‌اند؛ logic از A/G/C/L.

### WHAT CLAUDE MUST NOT INVENT

Phone canonical، rows silently merged، logical Case=entire batch یا MIS row globally؛ unrestricted repair/recall/financial relink؛ Finance/HR/Admin/Seller authority؛ cancelled/rejected release؛ fake zero/all-success؛ event range as historical-as-of؛ plan approval=delivery؛ preview guaranteed pure-read؛ forecast/quotas/SLA؛ new WP capabilities/schema.

### SHARED COMPONENTS TO REUSE

Scope/identity contracts، Case/Source context، custody/actor history، guarded action outcome، report filter/time/trust metadata، invoice/payment/review read context، common system states و partial result contracts. Shared visuals permissions را منتقل نمی‌کنند.

### MIS-SPECIFIC DESIGN NEEDS

Import Batch Console؛ Source Quality Explorer؛ Case Lineage Inspector؛ Custody Timeline؛ Reconciliation Workspace؛ Report Trust Context؛ Maintenance Boundary. هرکدام WHAT: provenance/quality/owner/proof/limits؛ layout/style/interaction visuals به Claude واگذار، طراحی یا پیاده‌سازی اکنون انجام نشده.

### LIKELY NEW SHARED COMPONENTS

Source/run context؛ lineage confidence/unknown context؛ per-item bulk outcome؛ metric coverage/gather window؛ cross-domain reconciliation evidence؛ maintenance impact boundary. reuse خارج MIS فقط با same scoped meaning/authority؛ no repair-all component یا backend architecture prescribed.

### DEFERRED / CONDITIONAL ITEMS

F01، resolver/OPD-01، readiness cause، F02/03/04 parity، Historical As-Of، large-batch retry/race، export fields/security، maintenance authority/retention، planning candidate/approval semantics، OPD-02/03/06/07/08. Visual treatment نباید unresolved authority یا data trust را نهایی/حل‌شده اعلام کند.

## 20. Freeze Review

| Review item | Result |
|---|---|
| Role / boundaries | source-quality-custody-reporting؛ Sales/Finance/HR/Admin اختیارات جدا |
| Current evidence | MIS#7، version2.0.123، 11 destinations و report sample؛ writes NOT LIVE VERIFIED |
| Product structure | 7 target domains با placement و guard distinctions؛ WHAT/WHY only |
| Gate0 / compatibility | INV-001…036 binding، KEEP + ADAPT، no destructive migration/ID/history/financial relink |
| Trust / safety | F01/F02/F03/F04/F09 explicit؛ missing≠0، per-item truth، active dependencies protected |
| Open decisions | section17 محدود و conditional؛ blocker برای read/product design ندارد؛ implementation dependencies باقی |
| Work performed | documentation و read-only business observation؛ report technical job auto-built، no business mutation intentionally executed |
| Stop | no implementation، Claude work یا HR Product Spec شروع نمی‌شود |

**MIS PRODUCT SPEC FROZEN — READY FOR CLAUDE REDESIGN**

**NO CODE CHANGED.**
