# Status Transition Catalog

رجیستری وضعیت‌های مسیرهای هشت نقش، supplement به Status Architecture گزارش اصلی. هر نام فقط در namespace خود معنا دارد. **CODE VERIFIED** برای enum/branchهای ارجاع‌شده؛ هیچ انتقال تغییردهنده اجرا نشده است. HR/Finance **NOT LIVE VERIFIED**. Terminal به معنی پایان مسیر عادی همان object است، نه حذف یا ممنوعیت هر استثنای آینده. `?` یعنی guards انتقال کامل/terminal بودن هنوز قرارداد قطعی ندارند و NEEDS VALIDATION؛ برای آن transition ساخته نشده است.

فهرست خام [تمام status occurrences استخراج‌شده](../evidence/status-occurrences-raw.json) پوشش جستجوی کل ZIP (شامل docs/tests/compatibility و ماژول‌های خارج هشت نقش) را می‌دهد؛ این فایل ۲۷۳۲ occurrence است نه ۲۷۳۲ وضعیت یا state machine. این catalog مدل محصولِ مسیرهای واقعی Audit است. enumهای customer portal، operations/shipping و campaign که consumer نقش‌های خارج دامنه‌اند به‌عنوان وابستگی نگه داشته شده‌اند؛ موفقیت runtime آن‌ها ادعا نمی‌شود.

## Seller Contact & Archive

مرجع `includes/class-sn-seller-flow.php:168,230,411,484` و invoice core/lead handlers. Actor=result owner با guard؛ صدور فاکتور event مستقل.

| Name | Module / Role | Meaning | Transition From | Transition To | Who Can Change | Terminal? | Temporary? |
|---|---|---|---|---|---|---|---|
| empty/unrecorded | seller flow / Seller | نتیجه ثبت نشده؛ unrecorded گزارش derived | source جدید/clear مجاز | no_answer/callback/duplicate/not_purchased/pre_invoice | owner/clear-policy/issuance | خیر | بله |
| no_answer | seller flow / Seller | پاسخ نداده با attempts | empty یا نتیجه قبلی مجاز | attempt بعدی، نتیجه جدید، archive بعد guard | owner؛ archive service | خیر | بله |
| callback | seller flow / Seller | تماس بعدی باcallback_at | نتیجه تماس مجاز | نتیجه بعدی/issuance | owner با زمان معتبر | خیر | بله |
| duplicate | seller flow / Seller | تکراری علامت‌خورده | نتیجه تماس مجاز | اصلاح مجاز طبقflow؛ ?پایان‌بودن کسب‌وکار | owner باguard | ? | وابسته به policy |
| not_purchased | seller flow / Seller | عدم خرید باreason | تماس مجاز | پیگیری/تغییر مجاز یاinvoice طبقguard | owner باreason | پایان نتیجه فعلی؛ globalterminal نیست | ممکن |
| pre_invoice | seller flow / Seller | فاکتور از پرونده صادر شده | پرونده فعال مجاز | paymentworkflow مستقل؛ archive overlay شرطی | invoice core، نه status دستی عادی | پایان صدور، نه پایان فروش | بله |
| archived_at set | archive overlay / Manager/Seller | پنهان‌سازی صف باreason | no_answer3+idle3day یاunpaidtype期限 | رفع archive درreconciliation/latepayment؛ ?restore دستی | cron/service، نه delete | خیر مطلق | بله/سیاستی |

Legacy lead_statusهای متنی «علاقه‌مند»، «در بررسی»، «کنسل»، «خرید کرده»، «پیش‌فاکتور شده» در selectهای قدیمی زنده دیده شدند. معنای enum و انتقال جاری آن‌ها باید با stage-one flow adapter تعیین شود؛ transition arbitrary برای labelهای legacy **NEEDS VALIDATION**. «خرید کرده» بدون paid evidence نباید completed financial محسوب شود.

## Invoice / Stage / Payment

مرجع `includes/class-sn-plugin.php:4819,6320,6497,6568,6577,30100,30382,30465,30515,30690` و `class-sn-dot-flow.php:6470`. Role display در Seller/Management/Finance؛ تغییر state عمدتاً service/payment actor یا Finance cap است.

| Name / namespace | Meaning | From | To | Changer | Terminal? | Temporary? |
|---|---|---|---|---|---|---|
| pre_invoice / invoice | صدور، مبلغ هنوز وصول نشده | creation | awaiting payment stage→review/partial/complete/cancel | issuance/payment flow | خیر | بله |
| partial_paid / invoice | پرداخت تأییدشده ناقص | approved/paid stage باremaining>0 | stage بعدی یاcancel/refundguard | finalizer/Finance | خیر | بله |
| receipt_uploaded / invoice mirror | مدرک واریز ثبت شده | pending/preinvoice/rejected correction | pending_financial_approval/approve/reject | receipt uploader/reconcile | خیر | بله |
| pending_financial_approval / invoice | مرحله درqueue مالی | receipt/gateway/reconcile | partial_paid/approved/paid/rejected/cancelled | Finance approve/rejectcaps | خیر | بله |
| approved / invoice | نتیجه مالی تأیید درmirror | pendingreview؛ remainingcompletion checks | normal complete؛ exceptional reopen/cancel | Finance/finalizer | پایان پرداخت عادی فقطاگرremaining0 | خیر درcomplete |
| paid / invoice | پرداخت نهایی درpath مربوط | verified/finalized payment | exceptionalreopen/cancel؛ normalcomplete | gateway/finalizer | پایان عادی باremaining0 | خیر درcomplete |
| rejected / invoice | رد مرحله باreason | currentreview | corrected receipt/resubmit/reopen یاcancel | Finance rejectcap؛ uploader بعد correction | خیر | بله |
| cancelled / invoice | لغو مالی | guarded active/partial،refundconfirmed اگرpaid | normalclosed؛ exceptionalpolicy ? | Finance rejectcap | بله برایcycle عادی | خیر |
| payment_archived / invoice | عدمپرداخت/obsolete overlay | unpaidexpired یاsupersededDot invoice | latepayment/revive طبقguard؛ restore ? | archive/flow service | وابسته بهguard؛ مطلق نیست | ممکن |
| recontact_requested / invoice | نیاز اقدام تماس مجدد | review/operations referral | submitted afterrecontact/close/cancel | workflow actor | خیر | بله |
| awaiting_payment / workflow | stageقابلپرداخت منتظرcustomer | creation/nextstageassignment | awaiting_financial_approval/completed/archive | invoice/paymentservice | خیر | بله |
| awaiting_financial_approval / workflow | منتظرreview | receipt/gateway evidence | completed/awaiting_assignment/rejected-return | Finance/finalizer | خیر | بله |
| awaiting_assignment / workflow | partialتأیید،nextresponsibility لازم | partialfinalization | awaiting_payment باstage بعدی | action assignment service | خیر | بله |
| completed / workflow | remaining≈0 وpaidcomplete | approved stage نهایی | exceptionalreopen/cancel/refund | finalizer؛ Finance exceptions | بله عادی | خیر |
| archived / workflow | cycle archived/cancelled | archive/cancel | guardedrevival درpaths؛ ? | service/Finance | وابسته بهreason | ممکن |
| pending / stage | مبلغstageتعریف‌شده | create stage | receipt_uploaded/pendingreview/paid/approved/cancel | issuer/paymentservice | خیر | بله |
| receipt_uploaded / stage | مدرکstage | pending/rejectedcorrective | pendingreview/approve/reject | uploader/reconcile | خیر | بله |
| pending_financial_approval / stage | صفreviewstage | pending/receipt | approved/paid/rejected/cancelled | review service | خیر | بله |
| approved / stage | مالی stage تأیید کرد | currentpendingreview | عادی ثابت؛ اثرcancel/reopen باaudit | Finance approve/finalizer | بله برایstage عادی | خیر |
| paid / stage | stageدرpathonlinepaid | successful verification/finalizer | exception reconciliation | paymentservice | بله برایstage عادی | خیر |
| rejected / stage | review ردشد | pendingreview/receipt | correction/resubmit طبقservice | Finance reject؛ uploader | خیر | بله |
| cancelled / stage | stagenonpaidلغو | pending/review درcancelinvoice | normalnone؛ exception ? | Finance cancel/service | بله عادی | خیر |
| pending / transaction | آغاز یا واریز pending | paymentcreate/manualsubmit | verified/approved/rejected/cancel | gateway/Finance/service | خیر | بله |
| verified / transaction | verification gateway | pending | Finance/stageapproval یاfaultreconcile | gatewayverify | خیر به‌معنیfinancialcompletion | بله/وابستهpath |
| approved / transaction | تأیید مالی واریز | pendingmanual | reconciliation/reversal exception | Finance finalizer | پایان تأیید عادی | خیر |
| paid / transaction-Dot | Dotpaymentتأیید | pending_finance | exceptionreversal/reconcile | Finance/flow | پایان عادی | خیر |
| rejected / transaction | رد سند/پرداخت | pending/review | corrective جدید طبقservice | Finance | پایان attempt؛ نهinvoice | خیر برایattempt |
| superseded / Dottransaction | پرداخت قبلی جایگزین شد | obsolete/noncurrent attempt | normalnone | Dotflow | بله attempt | خیر |
| pending_finance / Dottransaction | مدرک/پرداخت درصفFinance | pending/evidence | paid/rejected | Finance/flow | خیر | بله |

pending/draft/unpaid/empty aliases درlegacycountquery وجود دارند؛ معنی آن‌ها در source schema/runtime همیشه هم‌ارز pre_invoice نیست. normalize پیشنهادی به canonical resolver وابسته است؛ data rewrite فعلاً مجاز نیست. receipt وinvoice mirrors متعدد نیاز discrepancy diagnostics دارند.

## Financial Return / Repeat

| Name | Module / Role | Meaning | From → To | Changer | Terminal? / Temporary? |
|---|---|---|---|---|---|
| returned_to_seller | invoice return / Seller-Finance | مدرک/مرحله رد، اصلاح نزدSeller | review→reject→resubmit | Finance reject | خیر / بله |
| resent_after_return | invoice return | اطلاعات جدید بهreviewبرگشت | returned→new evidence→review | uploader/service | خیر / بله |
| cancelled_by_finance | return/repeat | cancelبدونpaidamount | active→cancel | Finance rejectcap | عادیبله / خیر |
| cancelled_by_finance_refund | return | cancelنیازمندrefundconfirmation | paid/partial→cancel | Finance guardedcancel | cancelterminal؛ refundexternalهنوزقابلپیگیری |
| payment_submitted_after_recontact | repeat metadata | مدرک بعدتماس واردreview | recontact→manualevidence→review | actionactor/service | خیر / بله |
| payment_completed | closeaction event | تکمیل،task بسته | opencompletiontask→closed | finalizer | بله task، نه حذفinvoice |

## Custody / MIS

مرجع `class-sn-plugin.php:19914,20244,20331,20731,20758,20945,22053,22353,22939,23208`. تغییر توسط actor مجاز با custody/recipient guard؛ اجرا در Audit انجام نشده.

| Name / object | Meaning | From | To | Changer | Terminal? / Temporary? |
|---|---|---|---|---|---|
| valid / MISrow validity | usableinput | importer validation | assignment/custody مستقل؛ validityedit ? | importer | پایان validation اینrow؛ نهفروش |
| invalid / MISrow validity | invalidinput | validation | correction/newimport policy ? | importer/repair | ? |
| duplicate / MISrow validity | repeatedinput | duplicateguard | repair/newimport policy ? | importer | ?؛ duplicateunlink خطرناک |
| unassigned / custodyderived | holderهنوزمعیننیست | validinput/return | assigned/prepare/directdelivery | MIS | خیر / بله |
| assigned_to_manager / staging | managerprimaryset | unassigned | pool/delivery/return | MIS | خیر / بله |
| distributed_forward / distribution | بهسطحبعدیتحویل | owned/returned | forward/delivered/return | eligibleholder | خیر / بله |
| delivered_to_seller / distribution | Sellerrecipient | pool/forward/direct | sourceconsumption/contact/invoice،guardedreturn | distributor/service | خیر؛ unusedبودن تضمیننیست |
| returned_to_manager / distribution | بهManagerبرگشت | priortransfer | redistribute | valid historicalactor | خیر / بله |
| returned_to_owner / distribution | بهownerقبلیبرگشت | priortransfer | redistribute | valid historicalactor | خیر / بله |
| returned_to_mis / distribution | برگشتمنبع | eligibleunused custody | reassignment | MIS | خیر / بله؛ F01guardgap |
| converted_to_lead / distribution | legacy leadmaterialized | eligibleitem | downstreamleadflow | conversionservice | terminal materialization؛ leadcycleمستقل |
| lead_created / compatibility | consumed/materializedalias | legacyconversionpaths | downstreamflow | service | ?exacttransition؛ aliasvalidation |
| converted_to_dot_case / distribution | Dotcasecreated | qualifyingreferralitem | Dotflow | Dotservice | terminalsourceconversion؛ Dotcycleمستقل |
| archived/cancelled/referral_cancelled / exclusionstates | returncandidate exclusion | workflow-specific | ?perdomainpolicy | service | NEEDS VALIDATION؛ queryexclusionبه‌تنهاییterminalproofنیست |

Materialized state تنها یکی از consumerproofهاست؛ V4invoicehardlink بدون تبدیلlegacy، «مصرف‌شده مالی» ایجاد می‌کند. statusonlyguard قرارداد کافی نیست. نقص P0 همین اختلاف است.

## Dot Case / Converter Contact

Enum labelهای کد `class-sn-dot-flow.php:5355,5365,6470`؛ case roles شامل Sellerassigned/Supervisor و converter خارجprimaryscope. انتقال‌های branchهای payment/assignment بررسی شده‌اند؛ برای چند label فقط catalog/renderer شاهد است و terminal کامل `?`، نه حدس.

| Case state | Meaning | From → To / Changer | Terminal? / Temporary? |
|---|---|---|---|
| awaiting_access_sms | منتظرSMSورود | created→selection بعدflow؛ service | خیر / بله؛ ارسالruntimeنشد |
| awaiting_customer_selection | منتظرانتخاب | accessflow→ready_for_conversion؛ customer/service | خیر / بله |
| ready_for_conversion | آمادهتبدیل | optionselected→assigned یاSellerownpath؛ directSupervisor/service | خیر / بله |
| assigned | converterمعین | ready→contact/payment؛ assignmentactor | خیر / بله |
| contacted | تماس انجامشد | contactupdate→nextcontact/payment؛ assignee | خیر / بله |
| customer_declined | فعلاًمنصرف | contactupdatewithreason→futurefollowup ? | مطلقterminalنیست / ممکن |
| payment_link_sent | linkcreated | validoption→pendingpayment/review؛ assignee/flow | خیر / بله |
| payment_rejected | پرداختردشده | Finance reject→correction/resubmit؛ Finance/flow | خیر / بله |
| deposit_paid | partialpaid | approvedpartial→remainingstage؛ Finance/flow | خیر / بله |
| completed | کلپرداختتکمیل | finalapproved→normalclosed؛ exceptional financialreopen | بله عادی / خیر |
| archived_unpaid_subscription | unpaidexpired | unpaiddeadline→archive؛ revivalpolicy ? | ? / سیاستی |

Contact states `new/no_answer/contacted/follow_up/confirmed/customer_declined/payment_link` درnamespace مستقل هستند. Meaningهای label: جدید/بی‌پاسخ/تماس/تماس‌بعدی/تأییدمشتری/انصراف/آماده‌لینک. Changer assignedactor باDotguard؛ follow_up نیازdue وdecline نیازreason دارد. Transition matrix کامل هرcontactlabel **NEEDS VALIDATION** درQAنقشconverter؛ terminallabel ازرنگ/نام تعییننشده است.

## HR

مرجع `class-sn-plugin.php:10190,10246,10397,10406,10439,19614`؛ runtime نقش **NOT LIVE VERIFIED**.

| State / object | Meaning | From → To | Changer | Terminal? / Temporary? |
|---|---|---|---|---|
| active / employment | همکاری فعال | workforceupdate→anyvalidatedemploymentstatus | HR | خیر / خیر تاupdate |
| inactive / employment | غیرفعال | HRchange→active/other | HR | خیر / سیاستی |
| suspended / employment | تعلیق | HRchange→active/terminated/other | HR | خیر / بله |
| resigned / employment | استعفا | HRupdate؛ downstream rehirepolicy ? | HR | پایانهمکاریمعین؛ rehirebusinesspolicy |
| terminated / employment | قطعهمکاری | approvedtermination→inactive+terminated؛ rehire ? | HRfinalapply/authorizededit | پایانemploymentcycle؛ login/runtimeguardNOTVERIFIED |
| probation / employment | آزمایشی | HRupdate→active/terminated/other | HR | خیر / بله |
| pending_review / request | بررسیوالد | created→nextpending_review/pending_hr/rejected؛ HRهممی‌تواندfinalapply | currentreviewer/HR | خیر / بله |
| pending_hr / request | تصمیمHR | chainapproved→approved/rejected/failed | HR | خیر / بله |
| approved / request | applyموفق | approvedfinal+successfulapply→normalnone | HRfinalreview | بله request / خیر |
| rejected / request | تصمیمرد | pending→rejected | currentreviewer/HR | بله request / خیر |
| failed / request | applyناموفق | finalapproveattempt→failed | HR/service | بله درcan_review فعلی؛ recoveryworkflowنیازvalidation |
| compensationactive/effectivedates | بازهحقوقمعتبر | priorclose/newinsert→nextinterval | HR | lifecycle temporal؛ salesstatusنیست |
| locked_after_payroll | علامتقفل | payload0؛ paidlockenforcement ? | HR/payrollservice ? | NEEDS VALIDATION؛ column وجود دارد، payrollworkflowثابتنیست |

Employment handler عمومی enumانتقالسخت مانندHRrequest ندارد؛ مجازبودن source→destination ازform alone استنتاج نمی‌شود. قطعهمکاری درrequest profileinactive/meta0 را می‌نویسد؛ نقشWP حذف نمی‌شود. login/sessionrevocation و unresolvedworkhandover نیازQA.

## Commission / Wallet / Report Job

مرجع plugin:33353,33400,35652؛purposecommission:26,584,593,675؛MISreportstore start/step/job/cancel/export؛ role-runtime Finance **NOT LIVE VERIFIED**.

| State / object | Meaning | From → To | Changer | Terminal? / Temporary? |
|---|---|---|---|---|
| generated / commissionrun approval | پیش‌نمایشساخته | dryrun→approved/rejected | financeviewguardedreview | خیر / بله |
| approved / commissionrun | قفل/تأییدinput | generated یاrejected→approved→posting | financeviewguarded APPROVE | reviewعادیterminal؛ posting مستقل |
| rejected / commissionrun | previewرد | generated→rejected؛ branchapprove current!=approved دوبارهapprovalرااجازه می‌دهد | financeviewguardedREJECT/APPROVE | **خیر مطلق** / ممکن |
| posted / runitem | walletlinked | eligibleitem→posted؛ idempotenthitrelink | posthandler | پایانعادیitem؛ reversal جدا |
| error / runitem | postfailed | attempt→error→retryeligible | posthandler | خیر / بله |
| skipped/existing / report result | already/ineligiblecounter | attemptclassification؛ objectstateهمهجا نیست | postservice | derived، terminalN/A |
| ready / previewrow | matchedamountpositive،postedنه | rules+paidinput→post candidate | derivedquery | خیر / بله |
| no_rule / previewdiagnostic | matchedrule/amountنیست | derivedquery→configchange | Financeconfig | diagnostic، terminalN/A |
| approved / wallettx | postedledgerrow | purposeposting→reversaldebitwhenexception | engine/service | originalcreditباقی؛ reversalseparate |
| purpose_commission_reversed / txtype | originalcreditreversed | purposecredit→reversal+typechange | cancellation/reopenhook | typeنهworkflowstatus؛ تاریخچهحفظ |
| credit/debit / direction | جهتledger | transactioncreation؛ deltaimmutablecontract | engine/Adminlegacy | statusنیست؛ balanceeffect |
| building / MISreportjob | جمع‌آوریchunked | start→ready/cancel/expiry | userjob/service | خیر / بله |
| ready / MISreportjob | snapshotقابلخواندن | buildingcomplete→export/cancel/expiry | reportservice | پایانbuild؛ عمرمحدود |
| cancelled / MISreportjob | jobلغو | activejob→cancel | authorizeduser | پایانjob / خیر |
| export ready / artifact | XLSXساخته | exportsteps→ready/expiry | authorizedreportservice | artifactstate، نهcasebusinessstate |

Normalization actions: KEEP namespaces وhistory؛ RENAME labels ازcanonicaldictionary؛ MERGE تنها aliases باpredicate/transition برابر؛ NEEDS VALIDATION برایlegacy/custom/terminalبدونguard. هیچstateباlabelمشابه بینcase/payment/run بهیکenum تبدیلنشود.
