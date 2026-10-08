# SENIOR SUPERVISOR PRODUCT SPEC

تاریخ: 2026-10-05 · محیط محصول: https://crm.maximumclub.ir/ · نسخه قابل مشاهده: **2.0.123** · نقش واقعی: **Senior Supervisor، حساب #4**.

## 1. Executive Summary

این سند WHAT / WHY نقش سرپرست ارشد را Freeze می‌کند: هماهنگی چند تیم زیر نظر Supervisor، شناسایی توقف‌ها، کنترل ظرفیت و کیفیت داده، توزیع موجودی مجاز، پایش تبدیل و فاکتور و مشارکت در زنجیره HR. اختیارات مالی، اعمال نهایی HR و مدیریت منبع MIS به این نقش منتقل نمی‌شوند. Visibility زیرمجموعه، write authority نمی‌سازد.

منابع الزام‌آور: [Audit پذیرفته‌شده](CRM-PRODUCT-ROLE-ARCHITECTURE-AUDIT.fa.md) (**A**)، [Gate 0](GATE-0-PRODUCT-INVARIANTS.md) (**G**)، [رجیستری کد](CODE-EVIDENCE-AND-ACTION-REGISTRY.md) (**C**)، [Status Catalog](STATUS-TRANSITIONS.fa.md) (**S**) و [Decisions Appendix](DECISIONS-VISIBILITY-INTERACTIONS.fa.md) (**D**). تمام INV-001…INV-036 در G الزام‌آورند؛ این Spec آنها را جایگزین یا تضعیف نمی‌کند.

Seller Design V1 Frozen و Supervisor Design V1 Frozen فقط بنیاد بصری و extension مشترک طراحی‌اند. هیچ prototype برای business logic، guard، KPI یا workflow استفاده نشده است. ساختار/نام component های واقعی آن دو در این مرحله بررسی نشده؛ reusable needs بخش 30 مفهومی‌اند، نه ادعای component آماده.

برچسب‌ها: LIVE VERIFIED = مشاهده واقعی همان صفحه/رفتار خواندنی در 5 اکتبر؛ CODE VERIFIED = کد ZIP Audit، نه تضمین byte های نصب‌شده؛ BOTH = توافق همان ادعا؛ INFERENCE = requirement محصول؛ NOT VERIFIED = اجرا/داده کافی بررسی نشده. UI action presence اثبات اجرای موفق یا مجوز endpoint نیست. تعداد داده کم، تست چندتیمی و اعمال writes را ثابت نمی‌کند. اطمینان inventory/اختلاف اعداد HIGH؛ runtime writes و enforcement منفی NOT VERIFIED.

Freeze مقصدهای هدف: Multi-Team Overview، Structure & Performance Explorer، Distribution، Ready Conversion نظارتی، Invoices، HR Requests، Reports، Own Account/Wallet؛ Case Explorer به‌عنوان contextual view با legacy facet؛ Personal Conversions شرطی. اهداف فروش، leaderboard و formula نرخ جدید تصویب نشده‌اند. KEEP + ADAPT پیش‌فرض؛ پیاده‌سازی همه توصیه‌ها خارج این مرحله است.

## 2. Role Mission

**مأموریت:** سرپرست ارشد مسئول هماهنگی چند تیم Supervisor در قلمرو مجاز خود، تشخیص توقف و اختلاف ظرفیت، توزیع موجودی مجاز و رساندن استثنا به مسئول عملیاتی/تصمیم‌گیر درست است؛ با پایش فاکتور، تبدیل، گزارش و درخواست HR، پاسخ‌گویی تیم‌ها را حفظ می‌کند.

Decision responsibilities: تشخیص اینکه کدام تیم به مداخله نیاز دارد؛ انتخاب مقصد eligible برای موجودی owned؛ تعیین اینکه اختلاف از scope/source/freshness است یا نیاز به reconcile دارد؛ بررسی درخواست HR فقط در جایگاه current reviewer؛ پیگیری صاحب next action. این نقش Seller operator، Finance reviewer، HR final authority، MIS owner یا صاحب همه lower-level actions نیست. فراوانی jobs پیشنهادی محصول است، نه نتیجه مصاحبه یا اندازه‌گیری کار روزانه. [A/D؛ INFERENCE]

## 3. Current Live Inventory

ورود واقعی از Supervisor #5 → «بازگشت به پنل HR» → Admin #1 → ردیف Senior #4 «مشاهده پنل». Header نقش و ID تأیید شد. بررسی بعد از تکمیل lazy loading انجام شد. سپس Admin بازگردانده و در plugins.php نسخه فعال Sales Network 2.0.123 خوانده شد؛ نصب/به‌روزرسانی/تنظیمات تغییر نکرد. شماره نسخه یکسان اثبات یکسانی build نیست.

| مقصد/بخش فعلی | مشاهده 2026-10-05 | Evidence / Confidence | تصمیم هدف و علت |
|---|---|---|---|
| ss-overview / نمای کلی | فیلتر تاریخ/فروشنده/status/code؛ sellers1 active1، leads0، invoices1، preinvoice0، total15000، paid0 | BOTH / HIGH؛ علت F02 از C | KEEP + ADAPT؛ operational overview با metric contract |
| ss-team / ساختار تیم | direct Seller0، indirect Seller1، direct Supervisor1؛ Seller#6 زیر Supervisor#5 | BOTH / HIGH | KEEP؛ current structure جدا از historical credit |
| ss-distribution / تخصیص شماره | owned0، eligible recipients2: Sup#5 و Seller#6؛ count/note/source/level؛ apply disabled؛ return count50 | BOTH / HIGH؛ apply NOT VERIFIED | KEEP؛ direct و skip-level صریح/مشروط |
| ss-sellers / عملکرد فروشنده‌ها | Seller فعال، lead0/invoice1/preinvoice0/amount15000/paid0؛ shared filters | BOTH / HIGH | MERGE view در performance explorer؛ facts محفوظ |
| ss-invoices / فاکتورها | یک preinvoice، stage1، due15000/paid0/remaining15000؛ facets و actions رسید/edit/resend/copy/history | BOTH / HIGH برای وجود؛ writes NOT VERIFIED | KEEP action-aware؛ مالی/فروش تفکیک |
| ss-repeat-actions / آماده‌های تبدیل | zero cases؛ متن صریح readonly و مسئول تخصیص direct Supervisor؛ selected/not-selected facets | BOTH / HIGH | RENAME presentation «آماده‌های تبدیل تیم‌ها»؛ internal ID حفظ |
| readonly ready queue | داخل ss-repeat-actions، مقصد جدا نیست | LIVE VERIFIED / HIGH | KEEP نظارتی؛ نه grant assignment |
| ss-converter-stats | zero converters/assigned/completed/amount؛ 0٪ با empty dataset | BOTH برای موجودی / HIGH | MERGE در explorer؛ نرخ NOT FINAL، empty ≠ underperformance |
| ss-my-conversions | zero owncases؛ توضیح فقط تخصیص به خود | BOTH برای shell / HIGH؛ receiving path NOT VERIFIED | CONDITIONAL MODULE — NOT ALWAYS VISIBLE؛ حذف به علت خالی ممنوع |
| ss-hr-requests | transfer/terminate scoped Seller#6، destination Sup#5، reason؛ empty requests | BOTH برای فرم / HIGH؛ review runtime NOT VERIFIED | KEEP؛ request/review ≠ final apply |
| ss-leads | filters، status summary و latest legacy leads؛ empty | BOTH / HIGH برای inventory؛ legacy-only CODE VERIFIED | KEEP + source-explicit adapter؛ «لیدهای قدیمی» baseline |
| ss-wallet | nav وجود دارد؛ در نخستین role landing بدون انتخاب عمدی wallet، empty history ظاهر شد | LIVE VERIFIED presence؛ deeper runtime NOT VERIFIED | KEEP personal؛ بازکردن مجدد عمدی انجام نشد؛ pure-read الزام G |
| report center | ده گزارش؛ time mode/role/person filters؛ invoices_register read: count1/15000، raw pre_invoice row | BOTH / HIGH shell و نمونه invoice؛ همه reports NOT VERIFIED | KEEP shared scope-aware Reports |

11 destination در nav، یک ready subsection و report center وجود دارند؛ آنها را 13 tab مستقل نشمارید. هیچ assignment/return/receipt/payment/HR review/export اجرا نشد. فقط tab navigation، financial history read و یک invoice report خوانده شد. [رکورد مشاهدات زنده این مرحله](../evidence/senior-reverification-2026-10-05.md) خلاصه قابل ردیابی بررسی است. تاریخچه مالی نمونه «هنوز سابقه‌ای ثبت نشده» و remaining15000 نشان داد؛ این شاهد عدم سابقه همان نمونه است، نه تضمین سیستم.

F02 هنوز در مشاهده جاری برقرار: پیش‌فاکتور row در invoice/report ولی zero count overview/performance. F03 query legacy-only از ZIP CODE VERIFIED؛ leads0 زنده موجود است، اما داده V4 امروز در پنل MIS/Seller مجدداً بررسی نشده؛ پوشش فعلی کل sources ادعا نمی‌شود. Gate0 F01 protection هدف الزام‌آور است؛ unsafe return اجرا نشده. بدون DB snapshot ادعای صفر write داخلی renderer نمی‌شود، ولی هیچ business mutation عمداً اجرا نشده است.

## 4. Senior vs Supervisor

| Capability / Concern | Supervisor | Senior Supervisor | Why Different? |
|---|---|---|---|
| Team scope | تیم عملیاتی خود | چند تیم زیر supervisors در scoped subtree | coordination بین تیم‌ها، نه همان جدول بزرگ‌تر |
| Direct sellers | recipient baseline روزانه | ممکن است direct Seller داشته باشد؛ نمونه0 | structure واقعی تعیین می‌کند؛ grant عمومی نیست |
| Indirect sellers | فقط scope مجاز خود | visibility aggregate/drilldown؛ write مستقل | hierarchy visibility اختیار نمی‌سازد |
| Multiple supervisors | daily mission ندارد | واحد اصلی تحلیل/پاسخ‌گویی | مسئولیت operational team حفظ |
| Distribution | owned→eligible direct Seller | owned→eligible Supervisor و مسیر Seller مشروط | primary route تیم، exceptional skip-level صریح |
| Skip-level allocation | از hierarchy بیشتر فرض نمی‌شود | technical path موجود؛ CONDITIONAL / NEEDS POLICY | حذف یا normalization خاموش ممنوع |
| Return | history/custody/eligibility scoped | reclaim انتقال‌های معتبر خود؛ نه recall تمام subtree | actor history با مقام بالاتر جایگزین نمی‌شود |
| Ready Conversion | actual assignment در observed direct flow | readonly aggregate | مسئول Supervisor همان فروشنده |
| Personal conversions | assigned-to-self طبق guard | conditional ownassignment/capability | senior mission supervisory؛ empty حذف نیست |
| Invoice assistance | actions scoped مشاهده‌شده؛ selection-link در نمونه Sup | receipt/edit/resend scoped controls؛ selection-link از Sup به ارث نمی‌برد | action-specific permissions نه role rank |
| Performance analysis | صف/تیم عملیاتی | team/supervisor/seller/conversion dimensions با facts مشترک | bottleneck چندتیمی |
| HR requests | subject scope و reviewerassigned | scoped subject + step review در chain | neither HR final apply |
| Reports | own team scope | چندتیم تحت scope و drilldown | source/time/field controls ثابت |
| Wallet | شخصی | شخصی | teamrevenue/commission دیگران نیست |
| Exceptions | رفع operational issue خود | coordination و escalation owner های تیم‌ها | ownership را از افراد نمی‌گیرد |
| Bulk actions | موجودی/actions واقعی خود | count-based allocation و return موجود؛ سایرها مشروط | کپی bulk grants از Sup ممنوع |

Evidence مرزهای موجود A/C و مشاهدات §3؛ ساختار هدف INFERENCE. رفتار Supervisor در این مرحله فقط header/nav کوتاه دیده شد، تحلیل تفصیلی آن از Audit پذیرفته‌شده است.

## 5. Senior vs Sales Manager

| Capability / Concern | Senior Supervisor | Sales Manager | Boundary |
|---|---|---|---|
| Organizational decision scope | subtree چند Supervisor خود | قلمرو شامل Senior ها و موانع وسیع‌تر فروش | scope سازمانی مستقل؛ Senior کل organization را نمی‌بیند |
| Archive review | context archive reason در case/invoice مجاز | archive explorer و types در Audit | archive management destination به Senior منتقل نشود |
| Extra-number requests | بررسی status مرتبط فقط در صورت حق موجود؛ request override ممکن | review/decision queue موجود | approve/reject Manager از سلسله‌مراتب به Senior نمی‌رسد |
| Cross-team distribution | بین eligible recipients زیر scope خود از own custody | Senior/Sup/Seller در قلمرو Manager | بیرون scope Senior escalation به Manager، نه write |
| Exception ownership | coordination exceptions تیم‌های خود | unresolved organizational/territory escalation | financial owner Finance؛ dataowner MIS حفظ |
| Reporting depth | تیم→Supervisor→Seller؛ facts/scopes | Senior-level/territory drilldown | همان contract؛ داده خارج scope ممنوع |
| Approval authority | HR assigned step؛ invoice assist مشروط | HR assigned step و extra-number review guards | Manager هم Finance خودکار نیست؛ هیچ blanket inheritance |

A Manager inventory مرجع این مقایسه است؛ role Manager امروز دوباره impersonate نشده. Spec هیچ وظیفه Manager را صرفاً به علت وجود internal endpoint مشترک به Senior اضافه نمی‌کند.

## 6. Daily / Occasional Jobs

| تناوب | Job / تصمیم لازم | Data Needed | نتیجه مجاز |
|---|---|---|---|
| Daily | تشخیص توقف چند تیم | current hierarchy، active staff، scoped workloads، next actor، stale/incomplete flags | drilldown/coordination؛ no invented target/ranking |
| Daily | هدایت موجودی خود | custody/source/eligible recipients/consumption | scoped distribution مطابق §11 |
| Daily | پایش invoice/ready exceptions | distinct invoice/stage/review، responsible Supervisor/Seller | read/assist دارای مجوز یا delegation |
| Daily | reconcile عملکرد غیرقابل اتکا | source coverage، query contract، old/new discrepancy | ثبت requirement reconcile، نه تغییر عدد/DB |
| Occasional | HR create/assigned review | subject، reason، destination، current reviewer، approvalpath | درخواست/step decision؛ final HR apply جدا |
| Occasional | reports/structure analysis | scope/cohort/time/source/freshness | read/export فقط مجاز و موردنیاز |
| Occasional | eligible return | own transfer proof + current dependencyguard | guarded reclaim، نه reset تاریخچه |
| Occasional | personal conversion/wallet | own assigned cases/own ledger | conditional personalwork/readonly account |

Data Not Needed: HR credentials/حقوق دیگران، بانکداری سازمانی، commission rules/posting، raw MISrepair، همه تماس‌های خارج scope، technical IDs به‌عنوان KPI. Actions Senior must not perform: مالی approve/reject/cancel/refund/post از صرف role، HR final apply/profile/role edit، MIS import/purge/repair، direct ready assignment در baseline نظارتی، contact write همه Sellers یا assignment خارج scope. [G INV-023…027]

## 7. Target Information Architecture

IA زیر مقصد و responsibility را مشخص می‌کند؛ شکل tabs/sidebar، layout، drawer و component style به Claude تعلق دارد. Current routes/IDs در اجرای فعلی باقی می‌مانند.

| سطح / گروه | مقصد هدف | محتوای موجود نگاشت‌شده | What / Why / شرط |
|---|---|---|---|
| Primary / Operational Management | Multi-Team Overview | ss-overview | current workload و استثنا؛ نه generic executive dashboard |
| Primary / Multi-Team Analysis | Structure & Performance Explorer | ss-team + ss-sellers + ss-converter-stats | structure و performance modes مجزا؛ sharedfilters/definitions بدون mixinggrains |
| Primary / Operational Management | Distribution & Eligible Return | ss-distribution | own pool و permitted paths؛ source/history explicit |
| Primary / Operational Management | Ready Conversion — Team View | ss-repeat-actions | readonly coordination؛ responsibility Sup |
| Primary / Operational Management | Invoices & Exceptions | ss-invoices | scoped invoice + stage/review و actionpolicy |
| Primary / Requests | HR Requests | ss-hr-requests | create/reviewassigned/status/history |
| Primary / Reports | Shared Reports | current report center | common contracts با multi-team scope |
| Secondary / Context | Case Explorer & Source History | ss-leads + read-through source adapters آینده | legacy facet baseline، fullcoverage فقط verifiedresolver |
| Secondary / Personal | Own Account / Wallet | ss-wallet + account actions | personal data نه teamfinancial |
| Conditional / Personal | My Assigned Conversions | ss-my-conversions | only actual eligibility یا own assigned cases؛ no permanent primary emptymodule |
| Advanced / Diagnostics | metric/source/link discrepancy detail | gaps F01–04/F09 | read diagnostic authorized؛ raw backend/repair tools در daily nav نباشد |

MERGE یعنی ادغام مقصد/فیلتر محصول، نه حذف داده، source tables یا permissions. اگر adapter آماده نیست، legacy destination با عنوان محدود باقی؛ مسیرهای بومی به hidden unsupported feature تبدیل نشوند. Savedviews/personalization P3 توصیه الزامی این Spec نیست.

## 8. Multi-Team Overview

Top level باید پاسخ دهد: کدام Supervisor و تیم مسئول است؟ workload اکنون کجاست؟ کدام work blocked است؟ آیا داده قابل اعتماد است؟ کدام invoice/ready/HR مورد نیاز مداخله است؟

Must contain: scope identity و evaluated_at، current direct Supervisors و active/total scoped Sellers، direct/indirect counts، own pool balance جدا از teamworkload، teamwise case custody، مالی/ready/HR exceptions با next actor، incomplete coverage و stale state. Team rows با drilldown به همان cohort و query contract؛ zero scope/emptydata با error متفاوت. اگر datafeed برای blocked/age موجود نیست «ناموجود/NOT FINAL»، نه ساخت count.

Shown metrics فقط dictionary §21؛ Primary proposed facts: staff counts، CaseWorkload، InvoiceCreated/OpenPreinvoice، ReadyQueueCount در صورت predicatevalidated، exception diagnostics. CompletionRate/targets/leaderboards/imbalance score **NOT FINAL — OPD-07 / metric decision required**. نمایش تفاوت workload برای تصمیم ظرفیت مجاز است؛ تعریف عددی «تیم ضعیف/عدم توازن» یا SLA از آن derive نشود. Revenue snapshot و collected stage amount با total invoice value مخلوط نشوند.

Acceptance آینده: انتخاب تیم subset scope می‌کند؛ cards/table/report same totals؛ filtered totals pagination-independent؛ stale/incomplete visible؛ no lower-level write authority از click inherited. [G INV-018…024/036]

## 9. Hierarchy / Structure

Current direct Supervisor relationship، direct Seller استثنایی و indirect Seller زیر Supervisor جدا نمایش داده شوند؛ هر person active/inactive و رابطه parent واقعی داشته باشد. Sample #4→#5→#6 فقط یک نمونه است، نه ساختار hardcoded. Scope = مجاز backend intersection با selection، نه تمام tree از dropdown.

Current hierarchy برای workload اکنون؛ historical attribution از event/snapshot هنگام assignment، sale یا credit. تغییر manager/position نباید original Seller، historical actor، commission owner یا نتیجه تاریخی را recompute کند. Missinghistoricalparent = UNKNOWN/NOT RECOVERABLE، نه parent امروز. [G INV-004/028/029]

Inactive profile در analysis/history قابل دیدن فقط scope/fieldallowed؛ recipient برای allocation باید فعال/eligible باشد. Structuralvisibility != permissiontowrite. Senior profile/parent/access را edit نمی‌کند؛ تغییر از HR request chain یا HR-authorized workflow. تحلیل چندتیمی واقعی با dataset یک تیم تست نشده؛ QA آینده باید direct/indirect/moved/inactive users را پوشش دهد.

## 10. Performance Explorer

محتوا: dimensions Team/Supervisor/Seller و separate Conversion mode؛ Invoice و CaseWorkload drilldowns با grain labels. فیلترهای مشترک scope، source، period/timebasis، employmentstate، statusnamespace؛ filter statusinvoice روی leads تنها اگر cohortrelationship واضح، نه reuse کنترل مبهم.

Seller dimension: active state، casecoverage، legacy-only count در baseline، invoicecreated، openpreinvoice، completed sales و valid collectedamount، lastinvoice timestamp با معنای روشن. Conversiondimension: assignedcase/ready/completed counts با predicate source، responsibleSup و converter؛ contact، receipt، stageapproved و completed جدا. No ranking یا combined score.

Metric ها یا §21 definition دارند یا **NOT FINAL — OPD-07 / metric decision required**. «نرخ تبدیل کل» فعلی با zeroassigned = «cohort خالی/داده کافی نیست»، نه 0% شکست؛ denominator/window نامعلوم = NOT FINAL حتی numerator موجود. Legacy lead count0 به معنی نبود workload V4 نیست. F02/F03: created/preinvoice/completed/totalcase/legacy-only پنج مفهوم مستقل‌اند؛ current wrongpredicate پشت نام جدید پنهان نشود.

قبول آینده: drilldown همان scope/cohort/status/timeversion؛ team subtotal های overlap دوبار grandtotal نشوند؛ one invoice attached multiplecases یک Invoice fact؛ multipletransfers یک Case fact؛ unverifiedalias aggregate incomplete. [G INV-001/002/018…022]

## 11. Distribution / Capacity

Baseline موجود: Senior owned pool را به recipients مجاز انتقال می‌دهد؛ UI نمونه Supervisor و indirect Seller هر دو را پیشنهاد می‌کند. CODE: nextrecipient از active chain، descendants و scopepositions؛ received custody، currentstep و referralhold می‌تواند eligibility را تغییر دهد. [C؛ plugin:23262،23374؛ LIVE §3]

Contract target: eligible recipient = active + correct position/path + allowed action + row scope + valid source/current state؛ visibility یا listedcandidate تضمین peritemapply نیست. هر item CaseRef/SourceRef/current custody، target/relationship، existing financial/dependencyproof، count/selection و reason را داشته باشد. primary destination Supervisor؛ مستقیم به Seller موجود silently removed نمی‌شود، **CONDITIONAL / NEEDS POLICY — SD-01**.

Skip-level exceptional path: recipient relationship و bypassedlevel واضح، actor reason/history محفوظ؛ reason اجباری target برای exception است، نه ادعای الزام فعلی handler. Policy approval/چه زمانی bypass مجاز است SD-01، تا تصویب newly normalized defaultgrant ندارد؛ existing workflow untouched. direct Seller واقعی با skippedSupervisor یکسان نیست.

Preview exactselection/amount-not-applicable/dependencies/recipient/state؛ apply actor/recipient/custody/scope/consumption دوباره check. Allocation eligibility باید case active financial dependency را به destructive redistribution/reset تبدیل نکند؛ انتقال مالی مجاز exceptional workflow نیاز policy جدا، نه reuse ordinary distribution. Atomic percase و partial perbatch؛ idempotent intent و audittransfer با original/creditowner untouched. Cap/count limits code current و target contract validation، نه capacity benchmark یا SLA. [G INV-004…010/034]

## 12. Return Rules

Senior **می‌تواند candidate reclaim به pool خود را initiate کند** فقط برای انتقالی که own actor history و guards آن را مجاز می‌کنند؛ code returnablequery join log.actor=actor و distributed_forward/delivered_to_seller دارد، current owner متفاوت و livelead/status exclusions؛ این predicate financial protection کافی نیست. مشاهده returncontrol != passguard. [plugin:20731،20758؛ C]

| نوع return | Senior target responsibility | شرط/دلیل |
|---|---|---|
| reclaim own priortransfer unused | conditional initiate | explicit permission + transfer proof/current scope/eligible dependency؛ G apply recheck |
| انتقال اجراشده توسط Supervisor بدون ownauthority | read/delegate Supervisor | rank اجازه recall همه تیم نیست |
| return/release منبع MIS | read/delegate MIS | MISguard/domain ownership؛ Senior owner منابع import نیست |
| legacy unassign | NEEDS VALIDATION، نه copy از Sup | وجود workflow نقش Senior اثبات کافی ندارد؛ adapter جدا |
| financiallyconsumed | ordinary return blocked | real legacy lead_id، V4followinvoice، flowevent/state، Dotfinancial links همگی resolve |
| cancelled-case release | Deferred OPD-03 | cancelled/rejected auto-unlink یا freepool نیست |

Blockedreasons: activeinvoice/stage/confirmedpayment، protectedmaterialization، fulfillmentdependency، current custodychanged، inactive/outscopeactor، noown transfer proof، unresolved/missinginvoice/link conflict. Response subject و reason/nextresponsibleactor را بدهد؛ ambiguous linkage failclosed با reconcile، نه phone-based guessing. preview allowed→invoicecreated قبل apply = safeconflict؛ concurrencycreation/return نباید orphan بسازد. History مالی/transfer/origin/credit حفظ؛ reset contactfield policy OPD-03. [G INV-006…010؛ QA-006…010]

## 13. Ready Conversion

TEAM READY VIEW: case/source، Seller، Supervisor مسئول، current custody، NextActor، conversion namespace/status، customerselection، assignedconverter، payment/stage خلاصه مجاز؛ groupbycurrent Supervisor/team. «انتخاب نکرده» facet موجود است ولی معادل ready approved یا completion نیست. age از timestamp معتبر آن event؛ updated_at را بدون تعریف waitingstart استفاده نکن؛ SLA جدید ندارد.

Senioraction: observe، scope-aware drilldown، شناخت responsible Supervisor؛ directconversion assignment در این module **NOT ALLOWED** در baseline. Escalate/route command فقط اگر workflow مستند موجود پشتیبانی کند؛ Audit evidence فقط supervision وجود دارد، dedicatedescalationcommand NOT VERIFIED → SG-02. نشان‌دادن صاحب کار مجاز است، ارسال/تغییر مسئول جدید اختراع نشود. Finance→Seller corrective nextaction با casecustody یکی نیست. [BOTH §3؛ G INV-004/005/023]

Acceptance آینده: teamread هیچ assignment command نمی‌دهد؛ hiddenassignment serverdenied؛ filterindividual بیرون scope emptyauthorizeddataset نمی‌سازد بلکه policydeny/selectioninvalid؛ nonexistentSupervisor unresolvedownerexception است، نه autoassignSenior.

## 14. Personal Conversions

**CONDITIONAL MODULE — NOT ALWAYS VISIBLE.** ss-my-conversions در current nav و empty ownqueue موجود است؛ خالی‌بودن دلیل حذف قابلیت نیست. visible اگر receiving capability واقعاً اجازه personalassignment می‌دهد **یا** currentuser own assigned cases دارد. وجود record حق جدید editing ایجاد نمی‌کند؛ view/history می‌تواند preserved باشد و action با guard جدا.

مسئولیت گرفتن conversion توسط Senior و receiving/selfassign path قطعی نشده: SD-02. تا تصمیم module در Personal/context قرار می‌گیرد، نه primary دائمی؛ directreadyassign role-level grant ندارد. Case personal با teamaggregate مشترک dataset بی‌label نباشد؛ seller/creditowner به Senior صرف assignee change منتقل نشود. Existing deep link/assignedhistory preserved، no reassignment/deletion. [A MISPLACED/NEEDS VALIDATION؛ G INV-004/023]

## 15. Invoices

Mustsee: invoiceID/code، scoped Seller، currentteam/Supervisor با basis، CaseRef/source/linkconfidence، invoice.status، stageID/no/state/due، paid/remaining/total/unit، FinanceReview/evidence/reason، CompletedSalepredicate و next actor. Phone searchattribute است، prooflink نیست. Historicalteam snapshot اگر ندارد unknown، notcurrent parentasfact.

| Action موجود/محتمل | Class | Senior contract / restriction | Evidence |
|---|---|---|---|
| list/filter/history/details | READ | scoped invoice/fields؛ namespace/ref واضح؛ observedhistory read verified | LIVE / CODE؛ runtime این نمونه readverified |
| copy payment link | ASSIST؛ data disclosure | فقط invoice مجاز، token حساس context؛ copy≠send، در این مرحله اجرا نشده | UI LIVE؛ execution NOT VERIFIED |
| resend payment link | WRITE / assisted operation | recipient/intent/channelguard، audit؛ noautomaticroleinheritance؛ never treatread | UI LIVE؛ write NOT VERIFIED |
| ثبت واریز و فیش | WRITE / ASSIST to Seller | explicit assistpermission + scopes + current stage/reviewguard؛ ثبت≠approval | BOTH presence/core guard؛ اجرا NOT VERIFIED |
| بارگذاری فیش | WRITE | evidenceappend/current stagevalid، field restrictions و filepolicy؛ receipt≠paid | UI LIVE/C؛ اجرا NOT VERIFIED |
| ویرایش پیش از پرداخت / prepayment | WRITE | only allowedfields/prepaymentstate؛ any existing paid/approved evidence guard؛ exacteditablefields NEEDS VALIDATION | UI LIVE؛ C stateguards، fullfield policy SD-03 |
| selection-link action مشابه Sup | NEEDS VALIDATION / NOT GRANTED | نمونه Senior control نداشت؛ fromSupinherit ممنوع | A observeddifference |
| Finance approve/reject/cancel/reopen/refund | NOT ALLOWED from Senior role | authority مالی مستقل؛ براین role target grant وجود ندارد | G/C Financeguards |
| invoiceissuance/manualcreation | CONDITIONAL / NEEDS VALIDATION | feature/override/sourceguard موجود اما Selleroperatorcorejob نیست؛ mainnavnewform اجباری نیست | A/C؛ SD-03 |

Assist یک purpose است، authorization نوعی read نیست؛ mutations explicitactionpermission می‌خواهند. Currentinternalpaymentguard Senior+scope را می‌پذیرد؛ target separatepermissions rolloutlater، currentcapreplace ندارد. Rejectstage history قبلی را حذف نمی‌کند؛ Senior next actorSeller را می‌بیند نه Financeapprovalbutton. Missingmirror/proof = incomplete/conflict. [G INV-011…017/023…025]

## 16. HR Requests

Create: transfer_seller/terminate_seller برای scoped descendantSeller با profile واقعی و reason؛ code cancreate positionSenior را می‌پذیرد. Destination transfer = Supervisor profile؛ UI نمونه Sup#5. Source targetsupervisorlist کد active Supervisors سازمان را می‌خواند؛ این را «همه destinations خارج scope قطعی مجازند» ننامید؛ subjectscope و final hierarchyguard جدا. Cross-team destination policy SD-04؛ no new immediate restriction یا grant از این Spec.

Review: فقط request.status=pending_review و current_reviewer_user_id=currentSenior، با actionpermission؛ Senior request های pending_hr را final apply نمی‌کند. Approvestep → nextreviewer طبق hierarchychain یا pending_hr؛ reject → rejected با actor/history/notes و targetreason requirement. FinalHRapproval + successfulapply → approved/applied_at؛ failedapply → failed، نه terminationdone. Intermediateapprove label باید «تأیید این مرحله/ارجاع بعدی» معنی دهد، نه «جابجایی انجام شد».

History: requestID/subject/oldparent/target/requester/reason/approvalpath/current reviewer/time/appliedat؛ hierarchy امروز path قبلی را rewrite نکند. Duplicatependingrequestguard حفظ. Enhancedterminationhandover OPD-05 Deferred؛ Senior reason/impactcontext را می‌بیند اما role/access/compensation/login تغییر نمی‌دهد. emptyqueue اجرای chainreview را تأیید نکرده است. [plugin:10284–10311،10338،10397،10439؛ CODE VERIFIED]

## 17. Legacy / Unified Case View

Target اکنون: **Unified Case Explorer به‌عنوان قرارداد read view** با stableCaseRef، sourcebadge و explicit Legacy-only facet. چون adapter کامل و coverage زنده تمام sources در این مرحله اثبات نشده، baseline قابل اتکا همان ss-leads با **«لیدهای قدیمی»** و scope/limitedlatestrowslabel است؛ fullunifiedcount ادعای فعلی نیست.

پذیرش view unified مشروط به G resolver، provenaliases و coverage/freshness؛ dualread beforecutover. source IDs/legacyrows باقی، IDs phonebasedmerge ممنوع، sourcefilter/querydomain visible؛ unresolvedlink diagnostics. Invoice/customer detail separategrains؛ legacyStatusrawpreserved mappedlabel با confidence. تصمیم target MERGE presentation + KEEPsource، نه DBmerge یا requirementmigration. OPD-01 unresolvedlineage Deferred؛ records ambiguous preserved. [F03؛ G INV-001…003/017/020/035]

## 18. Reports

SharedReports برای همه تیم‌های مجاز Senior و subset تیم/فرد؛ dropdown higherrole access بیرون scope ایجاد نمی‌کند. Querycontract شامل metricversion، scope intersection، source coverage، timebasis، cutoff/freshness، groupinggrain و drilldowncontext. CurrentSnapshot، EventOccurredInRange و HistoricalAsOf distinct؛ mode «وضعیت در لحظه» تنها با historyvalid می‌تواند historicalasof ادعا کند. MISreport eventrange+gatheredcurrent طبق G، asof محسوب نشود.

| Report موجود | محصول/معنای قراردادی | محدودیت |
|---|---|---|
| invoices_register | صدور invoice در بازه؛ pre_invoice included | نمونه count1/15000 LIVE؛ nofullreportqa |
| pre_invoices | historicalissuedpreinvoice | openpreinvoicecurrent metric جدا |
| sales | fullinvoicecompletionfacts | receipt/stageapproval alone نیست |
| online_sales / card_sales | valid successful/approved stages و uniqueinvoicecounts جدا | amounts/attemptdedup/currencyexplicit |
| customer_profiles / customer_invoice_details | customercontext/invoicedrilldown | phonegroup برای discovery، notCanonicalCase |
| mis_assignments | transfer events با actor/source | current custody≠historicaltransferactor |
| finance_pending / finance_rejected | queue یا reviewevents با modepredicate | readonly visibility؛ notFinancecommandauthority |

Exportavailability = conceptual export_reports + reportsourceguard + row/field policy؛ وجود reportselector exportgrant نیست. در این مرحله export اجرا نشده؛ export های supportedcode به runtime download تأییدشده تعبیر نشوند. Samecard/table/export totals تحت contract واحد، pagetotal مستقل. Sensitivebankfields/compensation نباید از export bypass شوند. [G INV-018…025/027]

## 19. Own Wallet

کیف پول Senior **شخصی** است: ownpostedcredits/debits و ownhistory/balance با meaning settlement اگر proof دارد. Teamrevenue، teamcommission، orgFinance و ledger دیگران نیست. Posted≠earnedpreview≠settlementready≠settled؛ تغییر current hierarchy credit تاریخی را تغییر ندهد.

F07 target: openingwallet هیچ credit/post/recalculate ندارد. در این جلسه wallettab عمداً انتخاب نشد؛ landing اولیه ناخواسته emptywalletdisplay داشت، پس ادعای «renderer wallet اصلاً اجرا نشده» نمی‌شود. Currentcode/engine dependencies از Audit مرجع؛ occurrenceposting NOT VERIFIED. Futurepure-readmigration ابتدا caller/dependency/shadow، سپس explicitcommand/event؛ حذف autopost بدون dependencyanalysis ممنوع. [G INV-026/031…033؛ A F07]

## 20. Exceptions

این دسته‌ها coordinationrequirements هستند، نه newescalationworkflow implemented. Owner کسب‌وکار و diagnosticEngineering جدا؛ SLA اختراع نشده.

| Exception / شاهد | Owner | Senior action | Escalation target | Resolution signal |
|---|---|---|---|---|
| workload variation / A distribution | respectiveSupervisor؛ own poolSenior | compareactualcounts/drilldown؛ permittedallocation | Manager برای outsideauthority | validcustody/newallocationhistory؛ imbalance threshold SD-05 |
| assignment conflict / C guards | commandactor/currentcustodian | inspectreason؛ re-evaluate بدون blindretry | Supervisor/Manager بسته به custody | coherentowner+committedtransferref |
| stale/incomplete metrics / F02–04 | data/reportowner با responsiblebusinessrole | markunavailable، inspectsource/cutoff | MIS/Engineering، Salesmetricowner | reconciledcontract/timestamp نه manualzero |
| consumed-return conflict / F01 | requestingactor؛ financial dependencyFinance | blockordinaryreturn، readproof | Finance/MIS برای reconcile؛ Supervisor operational | resolverproof و policyvalid؛ deletionlinkresolution نیست |
| invoice rejected/pending exception / C/S | Finance review یا Seller correction accordingnextaction | scopedread/allowedassist؛ superviseSup | Finance یا Seller/Sup طبق state | stage/reviewvalidnextstate+history |
| ready waiting forSupervisor / LIVE readonly | direct Supervisor | observeage اگر valid؛ identifyowner | responsibleSup؛ formalroutecommand SG-02 | validassignment توسط authorizedactor، notSeniorautoassign |
| inactive recipient / C HRactiveguard | distributionactor، HRprofileowner | rejectrecipient، inspectstructure | HR/Manager | eligibleactiveprofile یا anothereligibletarget |
| unresolvedsource/link / F01/F03 | MISsourceowner و Financefinancialcontext | failclosed؛ showunknown | MIS/Finance/Engineering | explicitproofmapping؛ phoneguess نه |
| HR request waitingchain / C/S | current reviewer | reviewonlyifassigned؛ otherwiseobserve | nextreviewer/HR | loggedstep decision/appliedat فقط finalsuccess |

Absentresolvedescalationowner = SG-02 PRODUCT GAP؛ «ارسال به مسئول» button جدید از این جدول مجاز نمی‌شود. Failureowner و next actor باید معلوم بمانند؛ current custody تغییر خودکار برای رفع labelexception ممنوع.

## 21. KPI Contract

تمام metrics metadata دارند: name/grain/source/cohort/time_basis/scope/numerator/denominator/status_definition/freshness؛ denominator counts=N/A،نه omitted. Defaults مشترک زیر برای ردیف‌های FINAL CONTRACT: scope = authorizedSenior subtree ∩ selectedteams/persons با field policy؛ freshness = evaluated_at/sourceversion/coveragewindow و stale indicator؛ اگر فراهم نیست data NOT AVAILABLE، نه zero. aggregationGrandtotal distinctfactkey است.

| Metric / وضعیت تعریف | Grain / source | Cohort / time_basis | Numerator / denominator / predicate |
|---|---|---|---|
| Invoice Created / Gdefined | Invoice/id facts | issuedinrange / creationevent | distinctissuedIDs / N/A؛ preinvoice included |
| Open Pre-invoices / Gdefined | Invoice/normalizedcurrentstate | scopedcurrent / CurrentSnapshot | distinctpreinvoiceactive / N/A؛ completed/cancel excluded |
| Pre-invoices Issued / Gdefined | Invoice issuance event | issuanceinrange / EventOccurredInRange | distinctissuedpreinvoiceIDs / N/A؛ laterpaidstillissued |
| Sales Completed / Gdefined、boundary OPD-06 | Invoice/validfinalization | completioninrange / completionevent | distinctfullpaidInvoices / N/A؛ approvedmirroralone insufficient |
| Verified Collected Amount / Gdefined، source reconcile required | validatedstage/paymentfact、chooseone no doublecount | valideventsinrange | sumvalidamountinunit / N/A؛ reversal separate |
| Total Cases / Gdefined、coverage conditional | canonicalCase / legacy+V4+declaredsources resolver | current scoped cases / CurrentSnapshot | distinctresolvedcanonicalIDs / N/A؛ unresolved/incomplete separate |
| Legacy Leads Only / Gdefined | reallegacyLeadID / sn_leads | explicitlegacycohort/currentquerycutoff | distinctlegacyIDs / N/A؛ NOT «کل لیدها» |
| Team Workload / Gdefined | Case/current custody + current hierarchy | ownedactivecases now | distinctownedactivecases / N/A؛ original owner نیست |
| Direct Supervisors / operationalcontext | Person/current parentrelation | scopeddirect Supervisor profiles now | distinctprofile/person refs / N/A؛ activefacetexplicit |
| Active / All Sellers / operationalcontext | Person/currentprofile + resolvedscope | direct/indirectscopedcurrent | distinctpersons matching active/all / N/A؛ employment/rawunknownexplicit |
| Own Pool Balance / conditionaldefinition | sourceadapter Case/custody | currentowned distributable pool | uniqueeligibleownedcases / N/A؛ legacy/v4 doublecount forbidden؛ fullresolver prerequisite |
| Ready Backlog / NOT FINAL — OPD-07 | DotCase/readyqueuepredicate | currentteamqueue، ready vs notselected separate | distinctqualifiedCase / N/A؛ predicate/cohort coverage SD-05 |
| Conversion Completed / NOT FINAL — OPD-07 | DotCase+validfullinvoiceproof | assignedcohort/window notapproved | counteligiblecompletedcases / N/A؛ window/scope source freeze prerequisite |
| Conversion Rate / NOT FINAL — OPD-07 | Casecohort/outcome | unresolvedassignedwindow | numeratorcompleted / eligibleassigneddenominator؛ zero denominator N/A |
| Blocked / Age / Overdue / NOT FINAL — OPD-07 | WorkItem/Event | validwaitingstart、duepolicy | distinctpredicateitems یا ageduration / N/A؛ noinventedSLA |
| Own Wallet Balance / ledgercontext | own wallettransactions | postedledgercurrentcutoff | approvedcredits minus validdebits byunit / N/A؛ settlement جدا، ledgerreconcile لازم |

Newmetric formula موسوم FINAL از این Spec ساخته نمی‌شود؛ Gdefined یعنی definitioncontract مصوب، نه implementation/valuespassed. Personcontext counts و ledgerformula فقط source-current context هستند؛ performance score نیستند. Timebasis در currentrates نامعلوم را publish 0% نکنید. F02compatibility: oldpreinvoice0/newcontract1 در samplereconcile، reasonpredicate؛ F03 oldlegacyleads0 در برابر unifiedcoverage فقط بعد resolverchecked. F04 علت build/cache/query هنوز validation؛ هیچ عدد را حدسی override نکنید.

Acceptance: cards/table/export یک version/scope/cohort/cutoff؛ querylimit totals را تغییرندهد؛ historical credit از currentmanager derive نشود؛ `old_total/new_total/difference/reason` قبل dashboardcutover. [G INV-018…022/029/036]

## 22. Permissions

نام‌ها **conceptual** هستند؛ هیچ WordPress capability ساخته/تغییر نمی‌شود. هر row نیاز Route+Module+Action+Row/Field serverguard؛ UI hiding is not authorization و nonce جای authorization نیست. Currentcap mapping futurecompatibility، viewwritegrant نه.

| Concept / action | Scope | State Guard | Row Guard | Field Restriction |
|---|---|---|---|---|
| view_multiteam / view_team_performance | authorizedsubtree∩filters | datasetcurrent/definedtime | permittedpersons/teamfacts | compensation/credentials hiddenunlessseparategrant |
| view_scoped_cases | declaredsourcescope | unknownresolutionexplicit | CaseRef/provenaliases authorized | phone/contact onlyrequiredcontext؛ nofinancialtokenbulk |
| assign_scoped_cases | own custody+eligiblepath | active/unconsumed ordinaryguard؛ preview/apply | actorcustody/recipientactive/scope/proof | allocationfields only؛ credit/origin/historyimmutable |
| return_eligible_cases | own transfer proof+currentscope | all financial links+fulfillment+materializationcheck | eligibleCase/current ownerrecheck | authorizedcurrentresetonly؛ historypreserved |
| view_ready_conversion | scopedteamaggregates | conversion predicate/sourceknown | responsibleSup/caseviewallowed | limitedcustomer/paymentcontext؛ noassigncommand |
| view_scoped_invoices | permittedinvoice/team | distinguishinvoice/stage/review | stableinvoice/caselink/contextguard | scopedcustomer/evidence؛ noorgbank/HRdata |
| assist_invoice_action | per-actionexplicitright+invoice scope | current stage/prepayment/reviewlocks | invoice/stage/assistedsubject authorized | alloweddelta/receiptfields؛ price/owner/history override notblanket |
| submit_hr_request | scopeddescendantSeller | validprofile/type/reason/duplicatependingcheck | subjectscope؛ targetSupervisorvalidated | requestedchangeonly؛ noaccess/credential/compensationedit |
| review_assigned_hr_request | assignedrequest | pending_review + current reviewer=currentSenior | subject/ref/current reviewerrecheck | step decision/reason/notes؛ nofinal apply |
| view_reports | source-specificscope intersection | selectedtimebasisvalid | facts authorizedwithallfilters | masking/sensitivefieldsprotected |
| export_reports | same queryscope + explicitexportgrant | validfreshcontract/exportstate | reportowner/sourceguards | approvedfields only؛ nohiddenfieldbypass |
| view_own_wallet | currentuserown ledger | readonlycurrenthistory | ownrecipientwalletrefs | nootherspay/financialsettings |
| view_personal_conversions / perform_assigned_conversion_action | selfassigned actualscope | conditional receivingeligibility/currentstate | actualassignee+Dotguards | allowedpersonalaction only؛ noallteamcontactedit |

Financeapprove/reject/post/rules/refund، HRfinal apply/changeaccess/credentials، MISrepair/import/purge و Managerextra-numberreview به این conceptualSeniorpolicy اضافه نشده‌اند. Existingtechnicalconditionalfunctions با inventory ثبت می‌مانند، SD-03 برای actionpolicy؛ هیچ currentpermission حذف یا write خارج scopegrant نمی‌شود. [G INV-023…025]

## 23. Handoffs

تمام domainhandoffs از G §5: correlation/subjectCaseRef یا HRsubjectref، from/toactor/role snapshot، type/reason/note/time، optionaldue طبق policy، state بومی، acceptedat فقط proof و failureowner. No genericACK/SLA جدید.

| Handoff | Contract / responsibility | Senior authority / limitation |
|---|---|---|
| Upstream → Senior custody/pool | upstreamactor/sourcehistory، current custodySenior؛ nextdistributionactor | readsown pool؛ sourceorigin/commission owner untouched |
| Senior → Supervisor | expliciteligible allocationevent؛ downstreamcustody/nextaction | assignowneligible case؛ originalactorhistorypreserved |
| Senior → Seller | directrelationship یا exceptional skip-levelrecord با reason | CONDITIONAL SD-01؛ nosilentdefaultbypass |
| Supervisor → Senior exception | responsible Supervisor و casecontext؛ escalationreason | monitorcoordination؛ command NOT VERIFIED SG-02 |
| Ready → Supervisor responsibility | direct SupervisorSeller relation و Dotstate | observe/drilldown؛ noSeniorassignmentgrant |
| Finance → Seller / team visibility | affectedstage/reason/nextactionSeller؛ Financeactorhistory | scopedmonitor/authorizedassist؛ nofinancialapprovalauthority |
| HR → Senior reviewer → nextreviewer/HR | current reviewerassignedstep؛ approved stepnotapplied | step decisiononly； final HR apply/outcome separate |
| Reports → reader | query/scope/cohort/freshness context | noownership/commission/customermutation |

Handoff تاریخی، current owner، next actor و creditowner هرکدام جدا persist/resolve؛ انتساب current hierarchy گذشته را rewrite نکند. Escalation در این جدول responsibilityrequirement است، newnotification/actionimplemented نه. [G INV-004/005/010/029]

## 24. Bulk Actions

| Candidate | Classification | What / Why / evidence |
|---|---|---|
| Bulk Assignment | SUPPORTED؛ حفظ آن REQUIRED در distribution؛ skip-level CONDITIONAL | countperrecipient/sourcefilter/multiselect در live؛ Cguards. Targetpreview/per-item result؛ التنفيذ NOT VERIFIED |
| Bulk Return | SUPPORTED technical؛ CONDITIONAL eligibility | count-basedreturn و selectedids handler؛ all financial links protection Gmandatory؛ نه successruntimeclaim |
| Bulk Escalation | NEEDS VALIDATION | existingcommand/reliablefailureowner یافتنشده؛ SG-02، newfeature تصویبنشده |
| Bulk Export | CONDITIONAL | reportexportsource/actionpermission و field policy؛ noexporttest، selectionsemanticsverifiedcode prerequisite |
| Bulk HR Review | NOT ALLOWED در baseline Spec | itemreviewcurrent reviewercode دارد؛ bulkSeniorauthority اثباتنشده؛ copyFinance/Supbulk ممنوع |

هر bulk operation selection/cohort ثابت، count/limits explicit، per-item guards/apply recheck، success/failed/skipped/alreadyapplied/unknown و committedrefs باید داشته باشد؛ partial≠allsuccess. Rate/SLA/performancecapacity این مرحله اختراع نشده؛ largebulk مستقبل QArequired. [G INV-034]

## 25. Data Visibility

**Must See:** scopedteams/directSup/activeprofiles، directindirectrelationship، CaseRef/source/current custody/next actor، eligible allocation/dependencyblockedreason، invoice/stage/review/remaining/unit، responsible Supervisorready، HRreviewer/path/reason، metriccoverage/time/freshness، historyproof.

**Nice to Have:** validatedtrends/cohorts، converterdrilldown، own walletcontext، scopedprofilesummary و history، contextualarchivereason، definedoverduedata. Targets/ranks NOT FINAL؛ وجود emptydata دلیل remove نیست.

**Should Hide / Restrict:** Financeapprovalcontrols، commissionposting/rules/rollout، HRcredentials، compensation بدون explicitauthorization، rawMISrepair، customersoutscope، technicalbackendraw statuses در defaultworkflow، UserIDsasKPIs، payment tokens/orgbankdetails غیرلازم. TechnicalIDs در authorizeddiagnostic/context ممکن‌اند، metricperformance نیستند. Businesslabels namespacequalified؛ rawunknownvalue فقط diagnostic با context. Hidepresentation به معنی serverpermission نیست؛ export و detail هم samefield policy. [G INV-016/017/023…025]

## 26. Error / System States

| State | Product requirement / نمونه |
|---|---|
| Loading | data هنوزنیامده؛ nozero placeholders که total واقعی تلقی شود؛ lazytabloading معنادار |
| Empty | queryauthorizedcomplete با nofacts؛ «هیچ پرونده owned نیست» با «هیچ فروش درکل CRM» فرق |
| No result | filtermatch ندارد؛ scope/cohort/filtercontext و resetfilter؛ noauthoritywiden |
| Unauthorized | action/row/field ممنوع؛ noleak sensitivecount؛ hiddencontrolserverguard |
| Stale | آخرین validcutoff و freshnesslabel؛ notsilentcurrenttotal |
| Incomplete | partialsource coverage، unknownidentity یا missingfeed؛ aggregatesnotfinal؛ no0%fake |
| Conflict | scope/custody/dependency/stage تغییرکرده؛ previewinvalid؛ refresh/reconcile قبل apply |
| Partial success | per-item committed/failed/skipped/unknown و counts؛ successfulitemsretry نکن |
| Retryable failure | transientidentified؛ sameintent/correlation و safecondition؛ permission/validationretryable نه |
| Outcome unknown | timeoutpossiblecommit؛ operationrefreconcile قبل retry؛ success/failureguess نه |

Metrics/performanceexplorer: contract/coverage/cutofferrors صریح؛ distribution: recipient/custody/consumedrace؛ reports: source/time/exportscope/incompletejobs. Zero فقط validcompletequeryvalue؛ nodata/unavailable مترادف نیستند. Readpaths businessmutation ندارند؛ temporaryreportjob metadata بهمعنی commercialcommit نیست. [G INV-018/021/026/027/034]

## 27. Product Gaps

تمام موارد این بخش **PRODUCT GAP — DO NOT IMPLEMENT** هستند؛ توصیه requirement به معنی featuregrant نیست.

| ID | Gap / شاهد | اثر / requirement | Priority / Confidence |
|---|---|---|---|
| SG-01 | F02 پیش‌فاکتور0 در برابر rowpreinvoice1؛ F03 legacy-onlyleadquery | metricdictionary/source coverage/reconcile قبل performanceuse | P1 / HIGH؛ currentF02LIVE، F03CODE |
| SG-02 | resolvedformalexception/escalationcommand و failureowner کامل NOT VERIFIED | observeownerbaseline؛ activationworkflow نیاز policy/evidence | P2 / MEDIUM؛ absenceofproof≠proofabsence |
| SG-03 | personalreceivingresponsibility نامشخص | conditionalmodule؛ noemptyremove/noassignmentinheritance | P2 / MEDIUM |
| SG-04 | skip-levelbusinessauthority/reasonpolicy نامشخص | pathpreserved و CONDITIONAL SD-01 | P1 / HIGHtechnical؛ policyunknown |
| SG-05 | cohort/window/freshness/waitingstart بعضی KPI ها ناقص | NOT FINAL / OPD-07؛ no0%falseevaluation | P1 / HIGHdefinitiongap |
| SG-06 | unifiedCase coverage/aliasproof کامل integration آمادهثابتنیست | contextuallegacyexplicit، adapterfuture | P1 / MEDIUM |
| SG-07 | currentUI/internalinvoiceactions و field policy نیاز reconcile | assistwritesexplicit، notFinanceauthority | P1 / MEDIUM |
| SG-08 | all-linkreturn/applyrace/historyprotection هنوز implementationverified نیست | G F01invariantsbinding؛ nowe unsafeactiontest | P0 / HIGHcodegap، runtimeNOT VERIFIED |

اولویت‌های فوق implementation راه‌اندازی نمی‌کنند. NumericSLAs/newqueues/newchannels/leaderboard/compensationanalytics از gap ها تولیدنشوند.

## 28. Deferred Decisions

| ID | Decision / مرجع | Safe baseline تا تصمیم | Owner / طراحی را می‌بندد؟ |
|---|---|---|---|
| SD-01 | skip-levelSenior→Seller authority/conditions/exceptionreview؛ extension OPD-10 | technicalpathpreserved؛ CONDITIONAL، noautoeligibility از hierarchy | Sales + ProductOwner + MIS؛ NO، activation مسیر depends |
| SD-02 | personal conversionreceiving/selfassignment responsibility؛ SG-03/OPD-10 | conditionalownmodule؛ assignedhistoryvisiblewithguard، noassignmentgrant | Sales + ProductOwner؛ NO |
| SD-03 | exactinvoiceassistfields/issuancerights/selectionlink differences؛ OPD-10 | per-actionexistinginventory + explicitguard؛ noFinance or blanketnewwrites | Sales + Finance + ProductOwner؛ NO، expandedactionsdeferred |
| SD-04 | HR transfer destination cross-scope/noopvalidation | subjectscopedrequest، targetvalidated، final HR apply؛ noimplicitfinalauthority | HR + Sales + ProductOwner؛ NO |
| SD-05 | ready/blocked/age/cohort/rate/imbalance threshold؛ OPD-07 | countswithsourcepredicatevalidated؛ rates/ageNOT FINAL، noSLA | Sales + MIS + ProductOwner؛ NO |
| G OPD-01 | uncertaincanonicalaliases | unknownpreserved/failclosed؛ noforcedmerge | Gowners؛ NO |
| G OPD-02 | recipientACK/SLA | commit≠humanACK؛ onlyexistingdecisionack | Gowners؛ NO |
| G OPD-03 | cancelledrelease/resetfields | ordinaryreturnblocked، historypreserved | Gowners؛ NO |
| G OPD-04/06 | refundproof/currencyrounding | readonlyexplicitproof/unit و boundaryconflict؛ nopolicyinvented | Finance + ProductOwner/Engineering؛ NO |
| G OPD-05 | enhancedterminationhandover | Seniorrequest/stepreview، nofinal apply | HR + Sales + Finance؛ NO |
| G OPD-08/09 | Finance separation/engines | outsideSeniorwrites؛ own walletreadtargetpure | Gowners؛ NO |

Deferred غیر blocker به معنی resolved نیست. Claude می‌تواند conditional/disabled/unknown contracts را طراحی کند اما business rule را قطعی نکند. CoreSeniormission/readonlyready/currentteamcoordination/guardeddistribution ثابت‌اند؛ unresolvedpolicy هیچ mainarchitecturecontradiction نمی‌سازد. FutureQA/implementationpending، designfreeze با backendcompleted اشتباه نشود.

## 29. Compatibility / Safe Adoption

**KEEP + ADAPT** برای همه recommendations. ExistingDataPreserved=YES؛ HistoricalIDsPreserved=YES؛ ExistingRelationshipsPreserved=YES؛ HistoricalAttributionPreserved=YES؛ MigrationRequiredNow=NO؛ Rollback=REQUIRED. Currentworkflow/permission بدون rollout تغییرنمی‌کند. No destructive DBmerge/statusrename/IDrewrite/invoicerelink/historybackfillguess.

| Recommendation | Current → Target | Compatibility bridge / Safe Adoption | Risk / شرط اجرای آینده |
|---|---|---|---|
| overview/performancegrouping | ss-overview/sellers/converterstats → multi-teamfacts/modes | KEEPsource/routes؛ ADAPTERFIRST، sharedquery contract، DUALREAD | MEDIUM Reports/Workflow؛ old/new/diff/reason و rollbackqueryversion |
| structure/history | current parent → current + historicalbasis | read-throughsnapshots؛ UNKNOWNpreserved | HIGH HistoricalData؛ norecomputepastcredit، negativeQA |
| distribution/return | currentrecipient/historyguard → all-linkeligibility/recheck | resolver تمام currentlinks؛ SHADOWVALIDATION→FEATUREFLAG | HIGH Database/Workflow؛ backup/snapshot/staging/raceQA/rollback؛ nofielddelete |
| readyconversionlabel | ss-repeat-actions → teamreadonlyready | displaydictionary؛ internalID/route/statevalues unchanged | LOW Workflow؛ noassignmentpermissionexpansion |
| personal conversionplacement | permanentemptytab → conditionalpersonal | keepdeeplink/assignedcases/rights؛ ADAPTERFIRST | MEDIUM Workflow؛ receivabilitypolicybeforeactivation، fallbacknavigation |
| legacy/unifiedview | legacyonly → unifiedlogicalsourcefacets | DUALREAD/provenresolver؛ legacyfilter/data retained | HIGH Reports/HistoricalData؛ unknowncountlabel、noDBmigration |
| invoiceassistance | sharedcurrentcontrols → per-actionexplicitpolicy | currentguardmapping + deniedendpointQA + gradualroleflag | HIGH Permissions/Workflow؛ currentcapsnotdeleted、audit/rollback |
| HR requests | currentreviewpath → explicitstep/final apply distinction | response/historyadapter، effectiveattributionpreserved | HIGH HistoricalData/Permissions؛ noforceprofile/accessrewrite |
| reports | multipleshells → commoncontext/metricdefinition | query contractversion/DUALREADreconciliation | MEDIUM Reports؛ cards/table/exportparitybeforecutover |
| wallet/purity | conditionalrendererpaths → ownpurebusinessread | callerdependencyanalysis、shadow、explicitcommand/event | HIGH Workflow/Integrations； noimmediateeffectremoval/noledgerrewrite |
| exception/errorcontracts | implicit/noop/partial → truthfulresponsibility/results | response/readadapter； commandonlywhenverifiedpolicy | MEDIUM Workflow؛ no newescalationgrant、unknownreconcile |

HIGH executionconditions: backup、snapshot、staging、rollbackplan、negative tests قبل implementation؛ هیچ‌کدام mutation اکنوننیست. Migration اگر بعداًلازمشد **FUTURE IMPLEMENTATION — REQUIRES QA / ROLLBACK**، Gate شرط immediateschema نیست. rollback نباید ledger/history پاک یا F01unsafeordinaryreturn پنهانی re-enable کند.

Acceptance contract آینده (NOT EXECUTED): G QA-001…036 binding؛ علاوه بر آنها actualSeniordeny directreadyassign/Finance/HRfinal/outscope؛ two Supervisor fixtures teamgroupdrilldown؛ movedSeller historicalattribution؛ same-phone cases remain distinct؛ activeV4followinvoiceblocked؛ preview/apply race؛ conditionalpersonalexistingdataaccess؛ receipt≠approval؛ partial bulk outcomes؛ repeated wallet reads create no ledger credit؛ old/new metric reconciliationو card/table/export parity. تست negative mutation فقط staging بااجازه implementation بعدی، نهمرحله Spec.

## 30. Claude Design Handoff

**WHAT CLAUDE MAY ASSUME:** مأموریت هماهنگی چند تیم؛ تفکیک روابط مستقیم و غیرمستقیم؛ CaseRef و SourceRef پایدار با وضعیت مجهول صریح؛ چهار مفهوم مالکیت مستقل؛ وضعیت‌های دارای namespace؛ مجوز مستقل هر action و محدودیت فیلدها؛ صف آماده تبدیل نظارتی؛ تعریف و تازگی مشخص metric؛ قفل مالی Return و بررسی دوباره هنگام Apply؛ ماژول شخصی مشروط؛ حفظ تاریخچه و سازگاری. اینها قرارداد هدف‌اند، نه API آماده یا اصلاح نصب‌شده.

**WHAT CLAUDE MUST NOT INVENT:** اختیار نوشتن از رتبه سازمانی، سیاست skip-level، تخصیص مستقیم آماده‌های تبدیل، حق دریافت conversion شخصی، تأیید مالی یا بازپرداخت، اعمال نهایی HR، فرمول metric یا cohort، هدف فروش، رتبه‌بندی یا SLA، وضعیت/انتقال جدید، کانال escalation، دستور ارسال اعلان، مقادیر تاریخی، migration یا قواعد و گیرنده پورسانت.

**SHARED COMPONENTS TO REUSE:** نیازهای مشترک مفهومی در Seller/Supervisor Frozen foundation: هویت نقش، جستجو و فیلتر scoped، جزئیات و تاریخچه entity، برچسب وضعیت از dictionary، حالت مجاز/غیرمجاز action، context فاکتور/مرحله/بررسی مالی، دلیل eligibility و نتیجه preview، قرارداد و تازگی گزارش، حالت خطا/نتیجه جزئی و حساب شخصی. Claude نام و variant دقیق را از library واقعی انتخاب می‌کند؛ این سند وجود component آماده را ادعا نمی‌کند.

**SENIOR-SPECIFIC DESIGN NEEDS:** مسئولیت و drilldown چندتیمی؛ تفکیک direct/indirect؛ مبنای جاری و تاریخی ساختار؛ حالت‌های performance با grain مشخص؛ context استثنای skip-level؛ ready نظارتی و Supervisor مسئول؛ تمایز بررسی میانی HR از اعمال نهایی؛ هماهنگی استثنا بدون اختیار نوشتن جدید.

**LIKELY NEW SHARED COMPONENTS:** نمایش scope و مبنای attribution؛ پوشش و reconciliation metric؛ source و اطمینان ارتباط؛ دلیل منع عملیات به‌علت dependency؛ مسئول اقدام بعدی؛ نتایج bulk به تفکیک item، شامل Outcome Unknown. اینها فقط نیازهای محصول‌اند؛ طراحی یا پیاده‌سازی component انجام نشده است.

**DEFERRED / CONDITIONAL ITEMS:** SD-01…05 و OPDهای Gate 0 در بخش 28؛ ماژول شخصی مشروط، سیاست مسیر skip-level، اختیارات assist اضافی، adapter پوشش یکپارچه، دستور escalation تأییدنشده و نرخ/age با تعریف NOT FINAL. طراحی می‌تواند baseline و این حالات را پوشش دهد؛ فعال‌سازی قابلیت تجاری به تصمیم و QA بعدی وابسته است.

دامنه Freeze، WHAT/WHY و قرارداد IA این سند است. هیچ کد، تنظیم مجوز یا داده تجاری عمداً تغییر داده نشد؛ بدون snapshot دیتابیس، درباره write داخلی renderer ادعای صفر تغییر نمی‌شود. اجرای writes، enforcement منفی، داده حجیم و گردش چندتیمی NOT VERIFIED باقی می‌مانند. Prototype مرجع منطق محصول نیست. هیچ UI طراحی نشده و Claude اجرا نشده است.

**SENIOR SUPERVISOR PRODUCT SPEC FROZEN — READY FOR CLAUDE REDESIGN**

**NO CODE CHANGED.**


