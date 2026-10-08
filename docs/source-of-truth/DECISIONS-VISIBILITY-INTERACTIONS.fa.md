# تصمیم، داده، ناوبری و کنترل خطا — ضمیمه Audit

این ضمیمه بخش‌های Decisions، Data Not Needed، Must See / Nice to Have / Should Hide، الگوهای interaction، scalability و error prevention درخواست اولیه را صریح می‌کند. پیشنهادها **INFERENCE** از موجودی و guardها هستند؛ محدودیت مشاهده زنده HR/Finance تغییر نمی‌کند. هیچ طراحی بصری یا permission تغییر نکرده است.

## Decisions / Data Visibility به تفکیک نقش

| نقش | تصمیم‌های اصلی | Must See | Nice to Have | Data Not Needed / Should Hide | Sensitive / Role-restricted | Confidence |
|---|---|---|---|---|---|---|
| Seller | چه کسی را اکنون تماس بگیرد؟ نتیجه/زمان بعدی چیست؟ چه محصول/مرحله مجاز است؟ چگونه رد یا مانده را پیگیری کند؟ | پرونده owned، source، تماس/attempt/due، مشتری، invoice stage/due/remaining، financial reason، next action | history تعامل، own commission rule match، نتیجه تبدیل شخصی | کل درآمد/حقوق تیم، برنامه توزیع MIS، ساختار مدیریتی گسترده، ابزار schema/repair، پرونده خارج scope | شماره/مدرک مشتری فقط پرونده مجاز؛ receipt در context؛ own wallet نه حقوق دیگران | HIGH ساختار؛ MEDIUM فراوانی/نیاز مصاحبه |
| Supervisor | به کدام Seller مستقیم تحویل دهد؟ کدام پرونده متوقف است؟ به کدام converter تخصیص دهد؟ چه HR request ثبت کند؟ | ظرفیت direct team، owned balance، نتیجه تماس، ready eligibility، invoice stage/owner/reason، transfer history | performance trend، own wallet، child profile summary | bulk HR credential، compensation دیگران، کل سازمان Finance، raw MIS repair، stage edit نامرتبط | scoped customers/invoices، reasonهای HR فقط subject/reviewer؛ commission شخصی | HIGH scope موجودی؛ MEDIUM سیاست delegation |
| Senior | کدام Supervisor/تیم نیاز به مداخله دارد؟ تخصیص به کدام سطح؟ کدام درخواست review شود؟ | hierarchy direct/indirect، team cohorts، exceptions، source-defined KPI، readonly ready queue، current responsibility | seller drilldown، scoped invoice assist، own conversion شرطی | تماس‌های own Seller بدون assignment، همه فرم‌های فروش در landing، receipt bank details نامرتبط، HR salary | team scope؛ HR request chain؛ own ledger | HIGH تفکیک نقش؛ MEDIUM personal conversion |
| Manager | کدام تیم کمبود/توقف دارد؟ شماره خارج صف پذیرفته شود؟ archive یا HR request چگونه پیگیری شود؟ | pool/team exceptions، invoice/reason، extra-number pending/conflict، archive reason/age، reviewer queue | customer timeline، drilldown stage، own earnings | UserID به‌عنوان KPI، فرم invoice Seller در landing، Finance approve/post، حقوق سازمان | current territory، customer/HR reasons؛ بدون blanket Finance rights | HIGH جز UserID قطعی؛ MEDIUM action delegation |
| Deputy | کدام لایه/مدیر مسئول انحراف است؟ ظرفیت کجا منتقل شود؟ exception کجا escalate شود؟ | governed totals/trend/cohort، hierarchy level، manager accountability، completion/remaining، critical exceptions | seller/invoice detail با drilldown، direct distribution exception، own wallet | چهار جدول طولانی همزمان، جزئیات تماس همه Sellerها در landing، عملیات receipt مستقیم، technical logs روزانه | aggregate scoped territory؛ sensitive detail فقط با نیاز و policy | HIGH معماری؛ MEDIUM targets هنوز مصوب نیست |
| MIS | کدام داده معتبر/تکراری/نامعتبر است؟ به چه گیرنده‌ای تحویل شود؟ آیا واقعاً مصرف/قفل شده؟ discrepancy چیست؟ | source file/batch/row، quality counts، custody/history، consumer hardlinks، valid balance، recipient active، report grain/time | paid outcome برای reconciliation، current hierarchy، run diagnostics | حقوق/credentials HR، جزئیات کارت بانکی مشتری، ویرایش مرحله پرداخت/قواعد پورسانت | phone/source provenance، گزارش scoped؛ advanced delete مستقل از read | HIGH source dependency؛ runtime actions NOT VERIFIED |
| HR | هویت فرد تطبیق دارد؟ سمت/مدیر/اختیار چه شود؟ انتقال/قطع نهایی شود؟ قرارداد اثرگذاری حقوق چگونه است؟ | user/profile identity، active status، legal parent، effective access، requested delta/reason/path، compensation interval/currency/rule | readiness counts، role impersonation، import diagnostics، structured employment status | customer commercial history عمومی، Finance receipt detail، daily sales operations، bank transfer controls | حقوق/هویت/credentials/export؛ صلاحیت read/edit/credential جدا پیشنهاد می‌شود | HIGH کد؛ role runtime NOT LIVE VERIFIED |
| Finance | مدرک کافی و متعلق به stage فعلی است؟ چه مبلغی تأیید/رد شود؟ لغو به refund نیاز دارد؟ run واقعاً مجاز و آماده posting است؟ | invoice ID/owner، current stage، due/paid/remaining/total/currency، receipt/ref/time، prior reviews، reason، selected run/approval/posting impact | sales hierarchy source برای reconcile، HR effective compensation inputs، gateway completeness | ویرایش قیمت/owner/گذشته پرداخت ضمن review عادی، HR bulk access editor، MIS distribution/repair، کل داده تماس بی‌ربط | receipt/card/gateway references وledger سازمانی؛ read reviewer writer post مستقل | HIGH کد؛ role runtime NOT LIVE VERIFIED |

«Should Hide» توصیه کاهش clutter یا field restriction است؛ اثبات این‌که UI فعلی چنین داده‌ای نشت می‌دهد نیست. Permission واقعی باید از backend و field policy اعمال شود؛ hiding ابزار امنیت کافی نیست.

## Navigation Audit — تعداد و grouping

| نقش | destinations فعلی | روزانه / عملیاتی | مدیریت / گزارش | account/advanced | پیشنهاد جایگاه داخلی و drawer/modal |
|---|---|---|---|---|---|
| Seller | ۷ تب زنده + new-invoice شرطی کد | leads،invoices،repeat | conversion شرطی،behavior،request | wallet وuser actions | customer/invoice detail contextual؛ stage/reason در workspace موجود frozen؛ tab مستقل دستی تنها مجازها |
| Supervisor | ۱۲ تب + shared report center | sellers،assign،ready،invoices | converterstats،HRrequests،behavior | wallet、logs、legacyunassign、subscriptions | subscription/receipt/history در detail؛ return با impact preview؛ Reports یک entry نه ده top-tab |
| Senior | ۱۱ تب + shared report center | overview、distribution、invoices | team、sellerperformance、readyreadonly、converterstats、HR | wallet、legacy leads、ownconversion شرطی | hierarchy drilldown؛ legacy source facet؛ ownconversion conditional |
| Manager | ۹ تب + shared report center | distribution、report、invoices、extra-number | overview、archives、HRrequests、behavior | wallet،diagnostics context | extra-number/HR review در queue با detail؛ archive explorer نه چهار top-level |
| Deputy | ۹ تب + shared report center | overview/exceptions | hierarchy + چهار performance view + invoices + Reports | distribution استثنا،wallet | performance level selector/drilldown؛ invoice details view؛ operational forms به entry شرطی |
| MIS | ۱۱ destination؛ mis-report link route مستقل | overview、import、assignment/return、pool | preview/MISreport、batches | quick、plan、livelead、logs/maintenance | picker batch/case مشترک؛ report detail readonly؛ destructive impact flow خارج گزارش |
| HR | ۱۲ تب در renderer؛ role-specific live ندارد | workforce、requests | compensation、hierarchy、structure、overview | manualadd、csv、bulk、positions、extra、logs | employee detail/history؛ sensitive access delta review؛ bulk/import مستقل از autosave |
| Finance | ۴ تب شرطی renderer + ۵ payment facet؛ role live ندارد | payments + evidence detail | overview、wallet、commission、gatewayreports | diagnostics、rollout/maintenance | invoice review detail؛ selectedstage/run confirmation؛ queue facets نه پنل‌های محصول جدا |

تعداد بر مبنای nav نقش است؛ nested facets، ده report type و user actions با tabهای اصلی جمع نشده‌اند. Shared shell باید هویت نقش و actor، همان زبان navigation و back/context را نگه دارد؛ تعداد تب یکسان الزام نیست. پیشنهاد grouping اطلاعات معماری است؛ هندسه drawer/modal و زیبایی به Claude تعلق دارد.

## Workflow Friction / Interaction Inventory

| نقش | مشاهده یا کد فعلی | مسئله / تصمیم | Evidence / Confidence |
|---|---|---|---|
| Seller | default صف بدون وضعیت خالی باallcount1؛ وضعیت‌های contact؛ invoice actions وwallet facets | empty state باید cohort فعلی را توضیح دهد؛ تماس/صدور/مالی در همان پرونده context وصل شود؛ repeated customer entry ازhardlink پر شود | LIVE برایصف/کنترل؛ CODE برایروابط / HIGH؛ تعدادclick/زمان کار اندازه‌گیری نشده |
| Supervisor | stats load دستی باهشدار504؛ loading کنارbehaviorloaded؛ ownconversion/ready/assign جدا | error/loading/data جدا؛ کار تیم وpersonalwork را جدا label؛ return/legacy unassign ورودی مشترک با semantics جدا | LIVE / HIGH؛ علت performance MEDIUM |
| Senior | status filter متن آزاد؛ readyreadonly ولیownconversionempty؛ lead legacy0 | dictionary انتخاب‌پذیر؛ restricted action explanation؛ source badge؛ conditional personalwork | BOTH / HIGH؛ نیاز ownwork MEDIUM |
| Manager | behaviorplaceholder باreview navigation؛ invoice UI عملیات محدودتر | dead-end candidate F05؛ available actions باpolicy توضیح؛ review queue وdetail بدون search دوباره | LIVE / MEDIUM علت؛ CODE intended renderer |
| Deputy | چهارجدول مشابه وhierarchy؛ invoice readonly | repeated search وswitch به level explorer؛ drilldown بهinvoice نهoperatingform؛ queryهای hidden eager | BOTH ساختار / HIGH؛ friction کمی NOT MEASURED |
| MIS | preview انتخاب batch را ازمسیر دیگر می‌خواهد؛ overview/health/returnability مخالفreport؛ Gregorianreportdates | مشترک picker/context؛ operational/reporting language جدا؛ canonicaleligibility؛ calendar/timebasis consistent | BOTH / HIGH؛ healthroot MEDIUM |
| HR | inline autosave،bulk/passwordmodal،import preview/apply،oldeditor | save ساده و access/compensation حساس یک interaction نباشند؛ preview exact affected IDs؛ formduplicate merge | CODE / HIGH؛ رفتار modal/save/error NOT LIVE VERIFIED |
| Finance | AJAXsearch/sort/page،receipt/detail،bulk100cap،rule/run/forms وconfirm words | pending-state locking؛ selection summary؛ peritempartialresults؛ preview≠post؛ criticalamounts همیشهدرcontext | CODE / HIGH؛ loading/toast/drawer/runtime NOT LIVE VERIFIED |

Notification/Help inventory: user actions وnighttoggle در live shell؛ Supervisor loading/error explanatory copy؛ MIS task guidance/health warning/report definitions؛ HR last-action/import reports وtechnicaldetails در کد؛ Finance flowguide/readiness/decisionresponses در کد. در Audit هیچ toast موفقیت mutation یا modal submit آزموده نشده؛ عدم مشاهده آن‌ها «missing» قطعی نیست. drawerها وprototype demo از UI واقعی به‌عنوان runtime proof استفاده نشده‌اند.

## KPI Decisions — KEEP / CHANGE / REMOVE / ADD

`CHANGE` این جدول معادل SIMPLIFY/RENAME یا اصلاح قرارداد metric است؛ تصمیم‌های feature اصلی همان vocabulary مورد درخواست را دارند. همه ADDها پیشنهاد INFERENCE، نه metric موجود.

| نقش | KPI فعلی / مصرف | تصمیم | دلیل و شرط پذیرش |
|---|---|---|---|
| Seller | total/followed/preinvoice | CHANGE | source/bازه/unique case وevent تعریف شود؛ clickdrilldown parity |
| Seller | approvedrevenue/successpayments | KEEP + CHANGE definition | stage vs invoice complete و مبلغ وصولی تفکیک؛ ownership ثابت |
| Seller | ownwallet balances | KEEP | posted/earned/settled را یکی نکند؛ readiness ازavailable جدا |
| Seller | due/overdue/blocked nextaction | ADD | actionability روزانه؛ دادهdue معتبر لازم |
| Supervisor | active/allSeller،assigned/free | KEEP + CHANGE | currentcapacity وsource/time؛ returnable financialguard |
| Supervisor | invoice0 stats/1detail | CHANGE | F04 قبل performanceuse reconcile؛ root cause QA |
| Supervisor | converters/cases/conversions/rate/revenue | KEEP + CHANGE | assignedcohort وdenominator؛ zero dataset نهunderperformance |
| Supervisor | team blocked/overdue/reviewage | ADD | مداخله actionable؛ نیازSLA مشخص |
| Senior | sellers/active/lead/invoice/preinvoice/paid/rejected/amounts | CHANGE | F02/F03؛ paymentamount وlegacycount contract |
| Senior | direct/indirect team counts | KEEP | structural context؛ revenue KPI نیست |
| Senior | team exception/cohort completion | ADD | شناخت تیم قابل مداخله؛ attribution اعلام |
| Manager | owned/assignedbyManager | KEEP + RENAME | currentcustody vs historicalactor؛ no mixing |
| Manager | UserID KPI | REMOVE | identifier performance نیست؛ header حفظ |
| Manager | report invoice metric | CHANGE | F04 detail reconciliation |
| Manager | pending extra-number/HR/archives age | ADD | approval responsibilities/exception actionability |
| Deputy | sellers/invoices/preinvoice/leads/amount/payment/reject | CHANGE | F02/F03 وancestor-level doublecount؛ governedfacts |
| Deputy | target/trend/variance | NEEDS VALIDATION→ADD | هدف مصوب،period وhistoricaldataset لازم؛ تزئینی نباشد |
| MIS | valid/waiting/withManager/ready/delivered/returnable | CHANGE | partition وoverlap روشن؛ returnable guard F01 |
| MIS | unique case/invoice、deliveryhistory、paid/remaining/discrepancy | KEEP | distinct grain؛ source link وcurrent/historical جدا |
| MIS | import lastreport/health readiness | CHANGE | run type/time/source؛ صفر ازstale schema کافی نیست |
| HR | role/level/profile counts | KEEP به‌عنوان context | productivity KPI نیست؛ نقص no-parent بهسمت حساس |
| HR | missingprofile/level/compensation/requests failed | CHANGE | role-conditioned completeness وage؛ runtime NOT LIVE VERIFIED |
| HR | onboarding/requestthroughput/age | ADD | نیاز job/time data؛ throughput کیفیت پذیرش را پنهان نکند |
| Finance | review queues/amounts | CHANGE | uniqueinvoice vs stages؛ currentqueue/periodentries؛ overlappingfacets |
| Finance | wallet positive/negative/pending/lastupdate | KEEP + CHANGE definition | ledger/allocation/settlement grain؛ posted!=earned |
| Finance | commissionrun eligible/excluded/errors/posted | KEEP | immutableinput/run/itemrefs؛ partial result واقعی |
| Finance | backlogage/reconcileexceptions/refundopen | ADD | owner/task/proof؛ role runtime NOT LIVE VERIFIED |

## Finance Error Prevention / Auditability

پیشنهاد محصول INFERENCE/HIGH بر پایه handlers؛ confirmationهای زیر requirement فعلی کاربر برای اجرای Audit نیستند. هیچ اقدام اجرا نشده است.

| اقدام | اطلاعات باید درcontext باقی بماند | confirmation / edit restriction پیشنهادی | کد فعلی / gap |
|---|---|---|---|
| approve stage | invoice/owner،stage#،due،paid/remaining/total/currency،receipt/ref、reviewhistory | خلاصه stage/moneyimpact؛ مؤلفه قیمت/owner/pastpaid ضمنreview immutable؛ metadata correction باdelta/note | pendingstageguard/finalizer موجود؛ metadata limitededit؛ no runtimeproof |
| bulk approve | selected count/IDs、sumofstageamount、exceptions | confirmation رویselectionfrozen؛ نتائج partialperitem؛ no silent successall | cap100 وapproved_ids/failed درکد؛ UI selection race NOT VERIFIED |
| reject | currentstage、reason、actor/time、nextSeller | reasonrequired؛ نمایش returneffect؛ settledstage blocked | transaction/FORUPDATE وreturned_to_seller موجود |
| reopen | previousrejection/archival،currentpaymenthistory،requested reason | explicit statebeforeafter؛ doublepayment prevention | approvecap guard؛ runtimeuntested |
| cancel | total/paid/remaining、Woo/Dot/commission dependency | impact summary؛ paidfunds نیازrefundproof؛ لغو معادلrefundبانکی نیست | refund_confirmed flag،cancel hooks؛ proofworkflow نیازvalidation |
| commission rule/toggle | scope/payment/employment/effectivedates/rate | affectedscopepreview؛ maker/checker policy اختیاریbusinessapproved؛ historicpost immutable | guard viewcap F06؛ runtimeuntested |
| review run | runID/inputtime/rules/HRsnapshot/totals/excluded | approve lockedinputs وreviewnote؛ rejectedrun می‌تواند طبقbranch فعلی دوبارهapproved شود | APPROVE/REJECT وlocked_at؛ approve شرطnotapproved دارد،rejected دوبارهقابلapprove |
| post/recalculate | selectedrun/eligibleitems/amount/existingtx/source | dedicatedwritecap،delta preview،safe retry；readers denied | flag+approved+APPLY وidempotencychecks؛ recalculate نیزviewcapwrite F06 |
| enablepostingflag | globalimpact/engine/rollback | rollout owner محدود؛ operationalviewer نداشته باشد | current viewcapF06؛ newcap پیشنهادی |
| export | fields/recipientpurpose،queryscope،sourcecompleteness | exportpermission مستقل؛ blankgatewayfields بهlocalfallback نسبت داده شود | CSVcolumns+nonce+finview؛ externalcompleteness runtimeuntested |

## Scalability Audit

این بخش CODE VERIFIED ساختار و INFERENCE risk؛ latency/load benchmarks اجرا نشده‌اند. هیچ عدد load capacity ادعا نمی‌شود.

| مسیر | evidence ساختار | محدودیت/ریسک | اولویت / پیشنهاد |
|---|---|---|---|
| Deputy | render همهperformancelevels/hiddentables | تکرارaggregatequery قبل نیازuser | P2 querysharing/lazybounded؛ benchmark پسقراردادmetrics |
| Supervisorstats | manualload وUI504 guidance | heavyfetch/timeoutrisk؛ stalezero ambiguity | P1 parity/freshness؛ P2 pagination/retry/queryprofile |
| HRworkforce/oldprofiles | boundedrowlist +clientfilter،separateglobalsearch | clientsearch فقطloadedsubset؛ exportall متفاوت | P2 serverpagination/filtercontract وscopeexplicit |
| HRexport/import | exportstream500batch،selected≤5000؛importUI5MB/10000rows | timeouts/partialfailure/identitymapping | P1 applyatomicity/partialreport؛ P2 jobprogress عند نیاز |
| Financequeue | page/limit10..100 default30؛bulk≤100 | selection stale/statechange race | P1 stage recheck؛ P2 queuepaging/freshness |
| commissionpost | selectedrunitems≤2000 درhandler | larger run processing coverage یاpartialremaining | P1 completion/parity review؛ stagedjob نیاز validation؛ no allitems successclaim |
| MISreport | chunk100case،max_id frozen،joblock،1hexpiry/max3jobs | changingdata طیجمع‌آوری،scope/custodyrevocation | P2 collectionwindowlabel/progress؛ preservejobhashchecks |
| MISquality/preview/rawlists | boundedlists وbreakdowns | visiblecount!=total،source mismatch | P1 canonicalsemanticcounts؛ P2 pagination/drilldown |
| Sharedreportexport | samequerycontract scoped | costlydatasets وPIIfieldpolicy | P1 scope/exportcap؛ P2 streaming/indexverify باbenchmark |

Architecture acceptance: index یاcache به‌تنهایی metric definition را درست نمی‌کند؛ اول predicates/grains/sourcecoverage وscope، سپس optimization با اندازه‌گیری. Browser screenshot اندازه performance نیست.
