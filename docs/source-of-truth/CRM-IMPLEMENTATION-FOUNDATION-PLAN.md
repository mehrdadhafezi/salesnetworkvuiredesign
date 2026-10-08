# CRM IMPLEMENTATION FOUNDATION PLAN V1

تاریخ: 2026-10-07 · **ENGINEERING ARCHITECTURE / MIGRATION-SAFE PLANNING ONLY**

این برنامه گذار تدریجی **KEEP + ADAPT** از پلاگین موجود به قرارداد هدف را تعریف می‌کند. مسیر پیش‌فرض **NO MIGRATION** است: حفظ DB، native IDs، source/financial links، history، routes/shortcodes، status namespaces، کاربران و permissions موجود؛ ساخت adapter/query contract و سپس wrap کردن actionهای حساس با validation. هیچ کد، schema، migration، role/capability، flag یا داده‌ای در این مرحله تغییر نمی‌کند. هیچ Git/GitHub setup یا UI design انجام نشده است.

منابع الزام‌آور: **X** [Cross-role Architecture V1](CRM-CROSS-ROLE-ARCHITECTURE-V1.md)، **G** [Gate 0](GATE-0-PRODUCT-INVARIANTS.md)، **A** [Audit](CRM-PRODUCT-ROLE-ARCHITECTURE-AUDIT.fa.md)، **C** [Code Registry](CODE-EVIDENCE-AND-ACTION-REGISTRY.md)، **S** [Status Catalog](STATUS-TRANSITIONS.fa.md). Role Specs: [Senior](SENIOR-SUPERVISOR-PRODUCT-SPEC.md)، [Manager](SALES-MANAGER-PRODUCT-SPEC.md)، [Deputy](SALES-DEPUTY-PRODUCT-SPEC.md)، [MIS](MIS-PRODUCT-SPEC.md)، [HR](HR-PRODUCT-SPEC.md)، [Finance](FINANCE-PRODUCT-SPEC.md). Seller/Supervisor business contracts از A/G/[D](DECISIONS-VISIBILITY-INTERACTIONS.fa.md) با تأیید کاربر در X؛ visual prototypes authority کد/مالی نیستند.

Evidence: code map **CODE VERIFIED** از snapshot پذیرفته‌شده و targeted static inspection؛ engineering proposal **INFERENCE / HIGH** در حدود invariants؛ installed bytes/runtime/negative/concurrency guarantees **NOT VERIFIED**. **HR ROLE RUNTIME NOT LIVE VERIFIED؛ FINANCE ROLE RUNTIME NOT LIVE VERIFIED.** هیچ live audit تکرار نشده؛ همه 248 فایل خوانده نشده‌اند. Runtime/DB/version compatibility deployment واقعی باید در future Phase 0 تأیید شود؛ صرف version=2.0.123 کافی نیست. [Evidence و validation این برنامه](../evidence/foundation-plan-validation-2026-10-07.md).

## 1. Current Code Map

Source snapshot: `audit-2026-09-30/source/sales-network-v4/`؛ workspace plugin موجود با phase1 changes تاریخچه خودش را دارد و revert نمی‌شود. Baseline code inspection را با deployment build یکی فرض نکنید.

| Surface | Current entry / owner | Coupling / foundation implication |
|---|---|---|
| Bootstrap | [sales-network.php](../source/sales-network-v4/sales-network.php): version/constants، always-loaded flows، light/full dispatch، activation/deactivation | SN_VERSION=2.0.123، runtime schema=2.0.122، build string مستقل؛ artifact provenance لازم؛ boot hooks side effects بررسی شوند |
| Lazy/full core | bootstrap load_core: migration/HR/hierarchy/scope/report/SMS/invoice/plugin؛ plugin_instance(run) | shortcode light invokes instance(false)، full context run registers actions؛ dependency/hook lifecycle نباید با wrapper دو بار اجرا شود |
| Routing/shortcodes | bootstrap shortcode map؛ [SN_Plugin](../source/sales-network-v4/includes/class-sn-plugin.php):325 register_shortcodes؛ portal registry :7506، URL resolver :7652؛ callbacks/template_redirect | stored page options/permalinks و legacy aliases حفظ؛ default slug لزوماً current URL نیست |
| Role panels | SN_Plugin Seller:13611، Sup:14922، Senior:12744، Manager:13556، Deputy:8333، MIS:11967، HR:10586، Finance:30813 | HTML/query/action controls و embedded domain logic در class بزرگ؛ role adapter به‌جای copy logic |
| Shared services | [Scope](../source/sales-network-v4/includes/class-sn-scope-service.php)، [HR Service](../source/sales-network-v4/includes/class-sn-hr-service.php)، [Hierarchy](../source/sales-network-v4/includes/class-sn-hierarchy-service.php) | constructors service graph؛ position fallback و current descendants؛ scope mode به معنی authority کامل نیست |
| Data access | `$wpdb` مستقیم در panels/handlers/services؛ table/column caches، option/meta stores؛ [Migration Service](../source/sales-network-v4/includes/class-sn-migration-service.php) | repository adapter روی existing queries؛ new path نباید installer/migrate/get-or-create را read فرض کند |
| AJAX/admin-post | SN_Plugin::run registrations؛ C handler registry؛ helper JSON و redirects | endpoints mixed payload/HTML/redirect semantics؛ preserve old clients؛ normalize new paths only |
| Reports | [Report Service](../source/sales-network-v4/includes/reports/class-sn-report-service.php)، [Executive](../source/sales-network-v4/includes/reports/class-sn-report-executive.php)؛ MIS [controller](../source/sales-network-v4/includes/reports/class-sn-mis-report.php) + source/model/store/xlsx | raw legacy query vs executive facts vs staged MIS jobs؛ common semantics، نه replacement تمام reportها |
| Invoice/payment | [SN_Invoice](../source/sales-network-v4/includes/class-sn-invoice.php):get/update/request/verify gateways؛ SN_Plugin invoice handlers/finalizer؛ [Dot Flow](../source/sales-network-v4/includes/class-sn-dot-flow.php) | financial mirrors/hooks، public tokens/callbacks، repeated attempts؛ DTO read نباید request_payment/finalize کند |
| Sales/customer flows | [Seller Flow](../source/sales-network-v4/includes/class-sn-seller-flow.php)، Dot/Product/Customer Portal/Operations/API modules از bootstrap | core hooks فراتر از 8 role panels؛ retain customer lifecycle/Complaint/Biawin routes؛ خارج scope redesign |
| HR | HR Service profile data؛ Hierarchy assign/end/history؛ SN_Plugin request/access/compensation/import/credential handlers؛ HR Transfer hooks | workforce write ممکن است role/access effect داشته باشد؛ transaction/recovery contract لازم، final apply≠intermediate approval |
| MIS | SN_Plugin batch/import/source assignment/pool/quick delivery/return/cleanup/planning؛ MIS Report source/store | source-manager assignment≠current custody؛ ingest repair/cleanup و report metadata writes آشکار |
| Finance | SN_Plugin can_view/approve/reject :29228…، review :30382…، Finance queries :36598… | F06 finance-view writes؛ optional wallet tabs cap مستقل؛ current guards را از نام label برداشت نکنید |
| Wallet/commission | wallet credit/recalc/render :35652…؛ legacy rule/run/post :33041…؛ [Purpose Commission](../source/sales-network-v4/includes/class-sn-purpose-commission.php) | stage hook engine + legacy APPLY coexist؛ immutable inputs/business key و no double execution اولویت |
| Shared utilities | [Helpers](../source/sales-network-v4/includes/class-sn-helpers.php):465 send_json؛ notification/SMS/gateway utilities؛ status/time labels | shared helper استفاده شود، اما side-effectful calls و output-clearing behavior برای new query path reuse کور نشود |

Duplication مهم: hierarchy/scoped sets میان renderers/handlers، legacy vs V4 Case links، preinvoice predicates، report totals، multiple stage/invoice/payment mirrors، چند نسل commission و HR update entryها. Foundation آنها را با یک semantic contract در adapters مقایسه می‌کند؛ حذف class یا replacement whole core پیش‌فرض نیست.

Hidden effects **CODE VERIFIED / conditional**: F07 wallet renderer→legacy autopost؛ bootstrap init ثبت report caps/schema و ایجاد missing page؛ admin_init schema/migration/cleanup؛ short `/i/...` handler اگر access_token خالی باشد آن را ذخیره می‌کند؛ HR get_or_create_profile write دارد؛ MIS import cleanup/schema helpers؛ dry-run run/options و report job/cache/cron metadata. Runtime occurrence هرکدام در نصب فعلی اثبات نشده. «تابع get/render» read purity guarantee نیست.

## 2. Target Layers

Logical boundaries، نه directory rewrite یا framework انتخاب‌شده:

| Layer | Contract / reuse |
|---|---|
| Presentation | approved design foundation را بعداً مصرف می‌کند؛ route/tab IDs و current old UI retain؛ no SQL/financial decisions in render |
| Role Adapter | viewer/role/position/field context و authorized actions از shared policy؛ module-specific DTO، no copied financial predicates |
| Application / Use Cases | single intent مثل scoped read، assign، review stage، HR apply؛ request correlation/result و commit recheck |
| Domain Services | proof-first identity/financial dependencies، invoice conservation، request transitions، entitlement ownership؛ business policy فقط G/X |
| Query Services | existing data فقط؛ grain/scope/unit/freshness/coverage؛ pure business reads، truthful unknown/partial |
| Repositories / Compatibility Adapters | `$wpdb`/meta/options/native refs با verified existing queries؛ explicit read vs write ports؛ prepared input/domain-qualified IDs |
| Legacy Bridge | current handlers/flows/hooks/token/page resolver reuse؛ legacy JSON/redirect/aliases unchanged؛ controlled single dispatch |
| Audit / Observability | common logical correlation/reference و domain extensions روی stores موجود؛ no compulsory new audit table |

Dependency direction: Presentation→Role Adapter→Use Case→query/domain + repositories؛ handlers/legacy bridge→same authorized use case وقتی path migrated؛ domain rules از HTML یا role title استخراج نشوند. Hook-based financial effects داخل write use case explicitly accounted؛ observer و real writer دو بار اجرا نشوند. Transaction/locking capability در repository execution port طراحی آینده validate می‌شود؛ generic wrapper به تنهایی atomicity نمی‌سازد.

## 3. Compatibility Strategy

Strategy labels ترکیبی و مرحله‌ای‌اند؛ DEPRECATE LATER نیاز usage/contract evidence و rollout جدا دارد، هیچ deletion اکنون.

| Shared contract | Strategy | Preserve / adaptation / acceptance |
|---|---|---|
| Case identity | KEEP native + WRAP + DUAL-READ + SHADOW | Legacy/MIS/V4/Dot domain refs؛ proof alias view؛ OPD-01 unknown؛ no immediate canonical ID persistence |
| SourceRef | KEEP + ADAPT | origin/native/batch/actor/time provenance؛ missing explicit؛ source assignment/custody separate |
| Ownership | WRAP + ADAPT + SHADOW | current/original/next/event/credit separate with provenance؛ no backfill current manager into past |
| Invoices/stages/payments | KEEP + WRAP query/write families | existing IDs/codes/tokens/statuses/mirrors/hooks؛ differences diagnostic؛ no relink/status rewrite |
| HR hierarchy/access | KEEP + query WRAP + temporal ADAPT | current assignments/legacy fallback؛ actual policy precedence unchanged؛ final apply guarded future |
| Reports/KPIs | KEEP + ADAPT + SHADOW | raw/executive/MIS sources named؛ metric contract parity، contextual legacy labels retained |
| Finance ledger/refund | KEEP + WRAP + SHADOW read reconciliation | original tx/payment/refund refs/history؛ compensating events not rewrite؛ OPD-04/06 dependent actions off |
| Wallet/commission | KEEP engines + ADAPT read + SHADOW calculations | never execute both writers in shadow؛ one approved route per entitlement OPD-09؛ legacy technical path remains until safe cutover |
| AJAX/routes/shortcodes | KEEP + versioned new adapters | old request shape/redirect/tab/deep links protected؛ replacement later only after client inventory |
| Repeated UI/query paths | DEPRECATE LATER candidate only | semantic parity/usage/rollback proven first؛ no removal because current list empty |
| Canonical alias persistence/durable cross-write intent store | MIGRATION CANDIDATE — NOT APPROVED if indispensable | defer until no-schema option proven insufficient؛ no ID rewrite or table merge |

DUAL-READ یعنی read existing representations؛ **dual-write فعال برای financial/HR effects ممنوع**. Shadow فقط pure query/evaluation یا isolated calculation؛ legacy handler را برای comparison اجرا نکند. Return guard target eligibility اختلاف مالی مثبت را blocker می‌داند؛ shadow outcome فعلی صرف مشاهده permission نمی‌دهد.

## 4. Read Foundation

1. inventory read call graph از entry/boot/constructor/hook تا query؛ label render کافی نیست. Reuse pure getters؛ get_or_create، schema ensure، finalize/credit/SMS را از new read path حذف یا به explicit approved maintenance/use case جدا کنید، پس از validation—not by silent code change now.
2. Read adapters روی native facts؛ normalized DTO/view model domain-qualified: subject/native/source refs، ownership dimensions، independent statuses، amounts/unit/basis، allowed fields/actions، metadata/error/coverage. DTO وجود object/proof را گزارش دهد؛ fields missing با UNKNOWN/NOT RECOVERABLE/N/A متمایز، صفر ساختگی نه.
3. Apply existing authorized scope **قبل retrieval/serialization**؛ field whitelist role-aware؛ API و cachekey viewer/scope/field policy لحاظ کنند؛ client hide حفاظت داده نیست. Current scope off fallback نباید new path را unconstrained کند یا implicitly usercap تغییر دهد.
4. Attach source/query contract version، evaluated/gathered time، pagination total vs loaded، bounded candidates، snapshot mode؛ step gather window اعلام شود. No cache keyed only by role؛ hierarchy/access change invalidates affected scoped derived results.
5. Compare old/new same cohort/scope/time against stable read fixture؛ expected semantic corrections جدا explain شوند. UI old available؛ sensitive data leak یا unexplained financial delta=no-go.

F07 gate برای new wallet/earnings reads پیش از Seller rollout هم لازم است؛ Finance screen phase7 بودن اجازه read mutation در phase2 نمی‌دهد. Renderer credit helper را query supplier استفاده نکنید. Token creation در short-link، init schema/cap/page seeding و service get-or-create نیز نیاز boot isolation/approved existing initialization analysis دارند؛ **plan NO DB CHANGE** با boot migration پنهان ناسازگار است.

Business pure read هدف؛ disclosed report-job/cache metadata طبق G مجاز technical write است فقط در future authorized flow. اگر full DB-read-only test لازم است، job persistence/cron/schema نیز isolated/disabled مسیر تست باشد؛ «business no mutation» با «zero SQL write» دو acceptance جدا هستند. این برنامه هیچ plugin bootstrap اجرا نکرده است.

## 5. Write Safety

Wrapper هر family باید auth/current scope، sanitized intent، nonce/CSRF طبق existing channel، fresh state/row version، dependency، permitted delta، idempotency relevant، audit و truthful per-effect outcome را اجرا کند. Legacy handler unchecked نمی‌تواند صرف outer validation atomic شود؛ critical fresh checks باید همان commit boundary writer باشند. Hooks/external calls partial effects حساب شوند.

| Family / existing seam | Required guard / execution accounting | Hold condition |
|---|---|---|
| Sales assign/return؛ C distribution handlers | actual actor/custody/eligible recipient/history row؛ union F01 dependencies؛ competing invoice intent serialized/rechecked؛ stable transfer proof | unresolved links، OPD-03/10 new rights، unsafe race |
| Invoice issuance/evidence/edit/reopen/cancel؛ invoice/Dot handlers | source-owner/action/field/stage state؛ prior payment/approval protected؛ amount/unit؛ preserve attempts/history، SMS/send separate effect | completion/release/refund/fields policy unresolved؛ no automatic replay external send |
| MIS import/source/return/plans | file/batch/run/native row scope؛ parser quality and consumed guard؛ cleanup/repair separately authorized؛ per-row limits/outcomes | financial or unknown protected source؛ schema health unknown؛ no bulk repair-all |
| HR profile/hierarchy/access/compensation/final apply | current request/reviewer/subject/target، field-sensitive authority؛ employment/access/handover/period impact؛ actual closed/inserted intervals & access effects | real HR runtime absent؛ OPD-05 or credential/access policy unresolved؛ unknown partial apply |
| Finance review | explicit approve/reject authority، current reviewable stage/evidence version، reason/allowed payment delta، paid/remaining checks، preserved previous approvals | stale/concurrent review، missing proof/unit/link؛ F06 separation not validated |
| Posting/wallet/reversal | approved immutable snapshot/routing/key/recipient historical input؛ unique business effect at commit؛ tx audit + hook reconciliation | OPD-08/09 unresolved، unknown prior outcome、no atomic duplicate protection |
| Maintenance/recalc/backfill/schema/repair | independent restricted policy、impact preview、protected all-link rows、explicit execution、audited recovery | new migration not approved؛ destructive rewrite/relink/reset； no safe rollback |

Future execution atomicity must match **actual storage engines/transaction support/multiple WP stores/hooks/external effects**. Do not promise single transaction covers SMS/gateway/session changes. If safe all-or-known-partial effect cannot be guaranteed, activation stays blocked؛ explicit reconciliation flow first. «success» helper return پس از unchecked DB call proof commit نیست (Hierarchy assign_parent مثال section10).

## 6. F01 Resolver

**FinancialDependencyResolver / Guard** logical shared component؛ implementation deferred. Input: domain-qualified Case/source/native refs + viewer/action/current-state context؛ invoice ID optional hint، phone never authoritative. Output: resolved relations/dependencies با evidence/domain/state/amount/unit، protected materialization/fulfillment، blocked reason codes، UNKNOWN/CONFLICT، coverage/version/evaluated_at؛ no permission grant.

| Adapter | Proof / limitation |
|---|---|
| Legacy lead→invoice | real `sn_invoices.lead_id` + actual native lead domain/record |
| V4 distribution→invoice | `follow_invoice_id` resolve existing invoice؛ lead_id=0 همچنان financial linked |
| Seller flow state/events | last_invoice_id + invoice event refs validated source context؛ last alone all links نیست، event alone active status نیست |
| Dot case/payment | existing case/invoice/payment relationships و guards؛ no phone join |
| Legacy virtual/compatibility ID | only validated encoding/domain + actual relation؛ fallback absence other links را negate نمی‌کند |
| Downstream/source obligations | materialized lead/Dot/source consumption/fulfillment/handoff constraints؛ nonfinancial نیز safety applies |

Adapter-first sequence: existing query extraction→proof classification/union→shadow old eligibility vs target→classified differences with affected native IDs→same resolver near commit in **all** ordinary return/reset/reassignment/cleanup callers. Policy union belongs G/X؛ role-specific duplicate predicates ممنوع. Do not call command from resolver؛ no record repair/relink. Cancelled release OPD-03 default blocked، refund proof OPD-04 independent.

Go/no-go F01: known V4 lead_id=0 + follow invoice must block؛ missing target/conflict block؛ Legacy/Dot/flow multiple links covered؛ preview then competing invoice/return and stale scope race preserve identity/history. No false allow accepted؛ false-block differences need approved release policy، نه bypass. Read adapter may ship earlier with limitation؛ protected command activation cannot.

## 7. Case Resolver

Sequence مطابق task: **Native refs → Alias adapter → Resolved logical view → Unknown/conflict → Shadow comparison**. No immediate canonical ID creation/table؛ resolved logical grouping reproducible from source proof، unresolved originals remain visible and separately counted.

Mapping includes native IDs across MIS row/batch distinction، distribution item، Legacy lead، Dot case، invoice relation—not force customer/user IDs into Case. Proof record: source relations، relation type، resolution basis/version/coverage؛ no phone/name/time heuristic financial joins. OPD-01 continuation vs related-distinct unresolved remains UNKNOWN. Canonical identity contract immutable، but persistent format/store decision deferred؛ logical reference must not claim invented historical identity.

Shadow cohorts capture native ref sets and explained alias mappings؛ compare source counts vs distinct resolved cases؛ cross-source coverage missing explicit؛ unknown does not get dedup guessed ID. Shadow alias view never writes aliases/relinks old invoices. Stored canonical persistence only if needed later، **MIGRATION CANDIDATE — NOT APPROVED** section14/23. Dependency resolver consumes proof result، no circular write or automatic materialization.

## 8. Permissions

| Layer | Map current guards to conceptual adapter | Compatibility boundary |
|---|---|---|
| Route | actual page options/permalinks، login/position/cap/shortcode entry | old routes/aliases retained؛ new adapter route reachability separate from action |
| Module | feature/tab wallet/HR/MIS/report gating | visible module≠all endpoint actions؛ no shell grant |
| Action | C handler policies، Finance approve/reject distinct، HR field-sensitive intents | evaluate server-side؛ nonce is not authority/idempotency |
| Row / Field | custody/owner/current reviewer/manager_id، HR subject scope، scoped invoice/report whitelist | row proof/hierarchy freshness؛ auth before DTO/cache serialization |

Current examples preserved: payment approve `sn_approve_payment`، reject/cancel `sn_reject_payment`، wallet tab `sn_view_wallet_commission`؛ matrix/backfill `sn_manage_wallets`/Admin، manual wallet adjustment Admin； HR position mapping/overrides allow/deny/inherit؛ Manager extra-number manager_id/current pending guard. Conceptual target policies from X not new WP capability names.

F06 writes `sn_view_finance` → future explicit action policy decision/evidence; F08 profile/position edit→role/access/compensation/credential effects need field/action segregation؛ **no cap rename/remap/reset now**. Future approved narrowing/expansion requires explicit access delta and role fixtures، never silently bundled with visual rollout. New wrapper may deny newly exposed dangerous action until policy approved؛ it does not authorize from view/rank.

Scope Service mode `off/audit_only/enforce_*` CODE VERIFIED؛ finance/Admin exceptions and position fallback mean reused service must be context-adapted؛ off does not remove existing ownership guards. Do not toggle enforcement option as rollout UI flag. WP capabilities should be checked on authorized actions، and nonce cannot replace authentication/authorization or replay prevention. [Official capability documentation](https://developer.wordpress.org/plugins/security/checking-user-capabilities/)، [Official nonce documentation](https://developer.wordpress.org/apis/security/nonces/). CRM hierarchy is separate business policy from WordPress role examples.

## 9. Reporting

Shared MetricQueryContract: metric ID/name، grain/source، cohort، time basis/mode/timezone، current vs historical attribution، scope/field policy، formula/predicate، unit/basis، freshness، coverage/version، paging/bounds. Existing Report Service/Executive/MIS source/store adapters remain; don't replace raw report clients or mix grains silently.

| Priority | Adapter / parity work planned | Acceptance |
|---|---|---|
| F02 | normalized open preinvoice predicate incl current pre_invoice، independent issuance metric | same scoped invoice IDs at fixed read snapshot؛ raw aliases not equated blindly |
| F03 | Legacy Leads Only honest source؛ separate proof-based Total Cases view | source coverage/resolved aliases/unresolved counts explicit؛ no V4=legacy count assertion |
| F04 | same grain/cohort/scope/time/source/version card vs list vs report | discrepancy explains provenance or blocks trusted metric؛ cache/build cause not guessed |
| F09 | event-selected cohort + gathered current status vs actual as-of | mode/time/coverage labels، stepped window not atomic snapshot； missing history prevents as-of claim |
| Finance parity | verified collection vs invoice totals vs ledger balances vs refunds/posting coverage | same unit/domain and tx refs، no UI-total ledger truth |

Shadow query only؛ report-job metadata/cache technical writes declared separate from business writes. Compare both query inputs and native-ID result sets، not merely equal totals؛ include filtered empty، source error، bounded/page total، multi-stage invoice and ancestor dedup. Fresh data needed for trusted drift diagnosis؛ historical attribution or cohort policy OPD-07 final rates blocked. Export uses same contract plus server-side field/scope/expiry/data owner gate؛ no test export now.

## 10. Temporal

Representation planned in view models: current parent/scope + provenance، historical actor/owner snapshots/validated intervals، event/effective/applied/posting/gather timestamps with timezone and coverage. Maintain existing dates/status IDs; presentation calendar conversion doesn't alter business cutoff.

Current targeted evidence: Hierarchy::assign_parent validates self/profile/cycle/current parent؛ closes prior relation then inserts assignment with current_time، logs and returns success without demonstrating transaction-wide checked effects in this branch. **CODE VERIFIED structure، atomicity/failure/race enforcement NOT VERIFIED.** WRAP future actual-effect verification، do not presume returned success assigned everything. Historical data may have missing intervals/roles؛ no guessed manager/rate backfill or scheduled application promise.

| Domain | Safe rule / missing evidence |
|---|---|
| Sales | workload current hierarchy؛ event/credit history preserved； recipient assignment doesn't rewrite original actor |
| HR | intended effect vs actual apply separate؛ interval overlap/close-insert/race/version test needed؛ handover OPD-05 |
| Finance | approved entitlement snapshot + historical recipient/share/rate؛ current HR changes cannot duplicate past entitlement؛ payment vs posting vs settlement time |
| MIS/Reports | import/source origin、delivery event、cohort time、gather window independent؛ Historical As-Of only reconstructed proof |

If true temporal integrity/durable scheduling needs additive persistence later، assess section14/23؛ no historical rewrite justified merely by a new timeline view.

## 11. Audit

Common logical reference support over current activity/history/run/item/tx/request stores: actor/acting-role/object-native-domain/action/before-after/reason/event-effective-applied time/result/correlation. Avoid forcing physical unified table؛ absent historical correlation stays UNKNOWN. New adapter correlation routes through domain audit where supported، response reference alone not durable evidence.

Extensions: Sales from/to transfer/eligibility/source؛ MIS file/batch/run/row/parser/quality/materialized refs؛ HR subject/profile/user/reviewer chain/access/compensation intended vs actual؛ Finance stage/evidence version/amount/unit/run/item/tx/business key/refund/reversal/retry. Secret/token/password/full bank/irrelevant PII logging forbidden. Test diagnostic hashes/scope/query metadata instead of raw receipts/customer rows.

Planned observability: source/contract drift categories، blocked reason counts، incomplete coverage/bounded tail، latency/query budget against measured baseline، write attempted vs committed per effect، idempotent hit/conflict/unknown، external correlation. Retention/access/payload policy must be approved before persistence extension؛ no invented numeric SLA/storage duration. Audit failure after possible commit yields known-effect + audit gap/unknown remainder، not zero-effect retry permission. Metrics themselves don't trigger repair/posting.

## 12. Idempotency

| Engineering outcome | Proof required / response meaning |
|---|---|
| Success | intended business effect verified committed/unchanged-idempotent؛ identity refs and coverage؛ transport 200 کافی نیست |
| Partial | some effects/items verified applied and others failed/rejected/skipped/unprocessed/unknown؛ exact per-effect breakdown |
| Failed | known rejected/failed execution outcome؛ if partial effects exist they remain explicit؛ no blanket zero-effect claim |
| Unknown | possible commit/external effect but unavailable proof؛ status uncertainty despite transport error؛ reconcile first |

Stable business intent/key distinct from request correlation، WP nonce، run ID، AJAX retry token. A new run/request ID must not mint a second entitlement for same underlying event. Snapshot recipient/share/purpose/source/stage/period as applicable؛ keys and role-share proof must prevent duplicate after HR/current-recipient changes، not rely solely on today's recipient. Exact noninvoice key OPD-09 deferred.

| Write family | Planned duplicate/retry proof |
|---|---|
| Finance posting | business entitlement key + approved snapshot + committed tx identity؛ atomic duplicate protection across concurrent handlers/engines؛ existing tx checks alone insufficient |
| MIS batch | stable source/run/row business intent؛ parser inserts/delivery refs + per-row actual outcomes؛ interrupted job retry missing eligible items only |
| HR final apply | stable request/change intent + fresh subject/hierarchy/access state + actual history/effects؛ no blind re-close/reinsert or role remap |
| Bulk | intent population snapshot + mutually exclusive final outcomes؛ requested=applied+skipped+rejected+failed+unknown when fully accounted؛ pending/unprocessed separate |
| External notifications/gateway/refund | authoritative external reference/proof؛ request correlation alone cannot enforce provider idempotency؛ unknown send/transfer held for reconcile |

Engineering must verify real DB engine/transaction/unique constraints/current record semantics before claiming atomicity. Existing stores may support reliable business refs without schema change؛ if durable intent uniqueness cannot be guaranteed, sensitive activation blocked and optional persistence decision section23 needed. **No transaction/unique index/schema change is approved or executed by this plan.** Provider retry/idempotency behavior not researched or promised here؛ touching it in implementation requires provider-specific official docs/contract.

## 13. API / Action Contract

Current SN_Helpers::send_json merges `success/message` and data, clears output buffers and calls wp_send_json؛ other flows redirects/HTML/nonce actions. Future new response envelope is versioned logical contract، not modification of every old response.

| Action family | Future treatment | Legacy compatibility |
|---|---|---|
| Existing scoped read AJAX | KEEP + query WRAP/ADAPT | preserve old action/nonce/payload؛ new consumer opts into normalized response |
| New read view model path | VERSIONED adapter after route/auth validation | transport AJAX/REST not selected yet؛ don't add new public endpoint automatically |
| Existing sensitive command | WRAP commit guards/use case incrementally | one actual writer؛ same old side effects accounted؛ old entry cannot bypass safety of newly certified action |
| admin-post forms | KEEP redirects/messages/tab context + internal normalized result | don't return JSON to expecting browser redirect |
| Reports/export jobs | ADAPT read/metadata/result contract | existing job/source/owner/expiry/download guards retain |
| Provider callbacks/token short links | KEEP + isolate/test later only if required | URLs/return shape/tokens/idempotency preserved؛ no cosmetic callback rewrite |
| Duplicated legacy paths | REPLACE LATER only with usage/parity/rollback evidence | no mass endpoint retirement now |

Future response fields: **ok، status، item outcomes، errors، unknown، correlation، freshness/version where relevant** plus authorized data/source/coverage and actual refs. `ok=true` only completed accepted intent (including proven idempotent unchanged)؛ partial/unknown explicit statuses and ok=false؛ distinguish transport success. `unknown` carries affected refs/count/cause safe for role، not secret/error stack. Items applied/skipped/rejected/failed/unknown and unprocessed distinguish؛ status/HTTP/error message alone not business truth.

New contracts should be machine-readable and stable؛ per-item sensitive details only within field scope. Retries keep same business intent and original correlation linkage؛ client-generated IDs never grant authority. Legacy clients retain prior schema until opt-in؛ compatibility tests catch response-field/HTML selector/deep-link changes. AJAX vs REST choice deferred rather than giant API rewrite.

## 14. DB Strategy

**Default NO MIGRATION؛ NO DB CHANGE.** Foundation logic/query DTOs can use current native records through adapters. DB schema parity inventory future read-only prerequisite؛ code includes dbDelta/migration/ensure hooks so deployment purity must audit automatic paths before upload/activation، not just proposed files.

| Candidate area | Classification | Guard / rationale |
|---|---|---|
| query/role/identity/source/state DTO adapters | NO DB CHANGE | computed view، existing IDs/columns/meta/options untouched |
| dependency resolver/metric predicates | NO DB CHANGE | validated existing links/read queries؛ no relink or normalization rewrite |
| logical correlation on existing domain logs | NO DB CHANGE if existing fields support | no historical fill/overwrite؛ durable support verify |
| report result materialization/read performance | DERIVED CACHE optional | authorized scope/contract/source version key، expiry/invalidation policy؛ disposable cache never ledger truth؛ needs future write approval |
| durable alias/intent/outcome/temporal facts absent from current stores | OPTIONAL ADDITIVE TABLE/COLUMN candidate | only justified by measured invariant need، separately approved persistence/version/index/privacy plan؛ not default requirement |
| transaction/business unique constraint not present | MIGRATION CANDIDATE — NOT APPROVED | atomicity evidence before deciding؛ real duplicate records cannot be auto-deleted to install uniqueness |
| table merge/ID rewrite/status rename/history rewrite/financial relink | MIGRATION CANDIDATE — NOT APPROVED; outside default scope | design consistency is no justification؛ separate explicit product/engineering decision required |

Do not invoke SN_Migration_Service::migrate، installer/schema repair CLI، activation/deactivation or runtime seed for planning/tests now. Future release must list every hook possible schema/cap/page mutation and prove no unapproved delta. WordPress documents that dbDelta compares/changes table structure and activation hook doesn't automatically run on plugin update؛ therefore update-time paths must be inventoried rather than assuming activation-only safety. [Official creating-tables guidance](https://developer.wordpress.org/plugins/creating-tables-with-plugins/).

Backup isn't authorization for destructive change؛ new columns/tables/flags/caches may be additive but still require separately reviewed future change. Snapshot restore or uninstall not routine rollback for financial/HR data.

## 15. Feature Flags

Flags are **proposed logical controls only**؛ no option names or implementations created. Routing switch and authorization separate؛ flags never grant rights. Keep old UI/routes while validation proceeds.

| Flag axis | Default / scope | Fallback |
|---|---|---|
| Role exposure | OFF؛ specific role plus authorized test cohort | legacy view on existing route |
| Screen/module | OFF independent of role shell؛ one bounded read module | old module/link with same context؛ no scope leak |
| Read path | legacy default→opt-in pure adapter after shadow | old read only if certified safe; unsafe side-effect path held/limited |
| Write path | OFF until family gates passed؛ independent of visual flag | safe legacy command only if same guard acceptance؛ otherwise disabled affected action + reconcile/escalate، not unsafe bypass |
| Shared presentation shell | OFF; approved foundation consumption later | old assets/layout available؛ no dual binding/duplicate commands |
| Shadow read/evaluation | OFF until purity/budget/privacy proven | disable shadow without changing canonical data |

Exposure rule: feature flag AND existing authorization AND source/readiness state—not flag alone. Compatibility matrix new UI×old/new read/write combinations certified explicitly؛ no silent new UI with unchecked legacy write. Current `sn_scope_enforcement_mode`، purpose-engine enabled flag and posting activation flag are existing policy/financial controls، **not UX rollout toggles**؛ do not repurpose or change.

Cutover proposal: shadow→approved comparison→scoped read exposure→broader validated read exposure؛ write activation separate owner approval/test evidence. Log version/cohort/active path nonsecret context، keep deep links/tab/query params، in-flight intents pinned to known handler version. Disable/rollback gates must not send uncertain financial write to another engine. New paths must not double-register legacy hooks/listeners؛ flags evaluated before writer dispatch، not after commit.

## 16. Rollback

Every future phase requires baseline artifact/hash + DB/options/caps/page/upload history snapshot where relevant، explicit flags، old route retained، reversible package deployment، no historical deletion، affected-domain reconciliation and rollback verification. None captured/executed against live site this task؛ local code hashes only.

| Phase family | Rollback contract |
|---|---|
| Query/adapter/shadow | disable exposure/shadow؛ restore previous package if compatible؛ evict only derived scoped cache؛ check legacy route/scope/count/history unchanged |
| Shared shell/read role UX | return old presentation/read path if safe، preserve IDs/tab context؛ ensure no duplicate event binding or hidden business writer |
| Sales/MIS writes | stop admission of affected new commands، reconcile in-flight/partial/current custody/native refs، then safe certified command routing؛ preserve transfer/source history |
| HR writes | code rollback doesn't undo personnel/access/session/credential effects؛ verify actual profile/hierarchy/caps and unresolved work، approved compensating action only； no auto role reset or guessed hierarchy reversal |
| Finance/posting/refund | quiesce affected admission، reconcile committed tx/business keys/run/item/refund/provider refs before engine switch/retry؛ code rollback must not debit/delete credits or repeat external transfer |

Never restore whole DB snapshot over post-snapshot legitimate business transactions as generic rollback. Recovery point/time and actual delta reconciliation/domain signoff needed before any exceptional restore. Valid financial corrections append linked reversal under policy؛ HR restore with effective/history/access audit. Backward-compatible package cannot overwrite historical state. Old UI availability not guarantee old command safety؛ F01/F06/F07 no-go can't be bypassed by fallback. Ledger/audit refs, changed user access، protected invoice links and unknown outcomes checked after rollback—not only homepage loads.

## 17. Phases

Role exposure order preserves user sequence؛ safety dependencies pulled forward. No schedules/person-days guessed؛ each phase has small PRs and independent gates. Read/write tracks separate، shared foundations may prepare code paths before role screen rollout without activating domain writes.

| Phase | Planned scope / dependency | Exit evidence / rollback |
|---|---|---|
| 0 — Foundation | confirm deployed artifact/PHP-WP-DB engines/hooks; existing routes/caps/native refs baseline؛ pure query seams؛ shared identity/F01/permission/outcome/audit contracts + shadow tests | no unapproved schema/cap/data delta؛ protected resolver scenarios and privacy/query purity evidence؛ flags OFF؛ revert package/shadow |
| 1 — Shared read models + approved shell integration | normalized scope/source/status/metric/time DTOs؛ consumed approved design foundations later، no UI redesign here | source-aware truthful partial/unknown؛ old routes/assets intact؛ parity/contract/negative read tests |
| 2 — Seller read UX | own authorized read context؛ product contract A/G/D؛ wallet F07 isolation and token/public call graph already gated | role smoke/read browser parity/no business mutation؛ writes stay legacy-certified or disabled unsafe extensions |
| 3 — Supervisor | direct team/ready oversight-assignment context، reports/request reads | no indirect privilege inheritance؛ F04 surfaced/reconciled for trusted metrics؛ F01 gate before return activation |
| 4 — Senior → Manager → Deputy | sequential cohorts، multiteam/governed report/context reuse؛ Manager-only extra-number authority، Deputy conditional paths | no rank grants، legacy facet/count coverage honest؛ step reviewer/owned allocation guards certified separately |
| 5 — MIS | source/quality/lineage/report adapters reuse Phase0 proof； diagnosis first، ingest/return/repair separate activation | fresh health/completeness، per-row partial/retry، F01 protected cleanup؛ unsafe maintenance OFF |
| 6 — HR | dedicated role/read fixtures first؛ current/effective/access context； granular writes after policy/actual-effect gates | real HR runtime + negative guard tests؛ OPD-05/temporal/compensation recovery before dependent final apply activation |
| 7 — Finance | dedicated role/read/ledger/run context first؛ review/post/refund separate staged activation | F06/F07、unit/proof/snapshot/engine/key/concurrency/unknown recovery gates؛ committed effect reconciliation on fallback |

**Dependency adjustment:** F01 guard/identity shared semantics، F07 pure wallet reads، permission/unknown result infrastructure enter Phase0/1—not deferred to MIS/Finance UI phases. F02/03/04 report adapters/parity needed before performance trust at Phase2–4؛ F09 historical claims stay disabled until history proof. HR eligibility/hierarchy read foundation needed for all sales scope earlier, even HR writes phase6. Finance review/payment DTO foundation earlier; real Finance operation rollout remains phase7 conditional. High-risk command gates may be validated separately when authorized، not bundled with shell redesign. No requirement all phases migrate DB.

## 18. High-Risk Gates

Go/no-go is evidence-based؛ unexplained protected financial/access difference blocks affected release، not converted to arbitrary acceptable percentage. Owners sign scoped validation، not blanket architecture approval.

| Gate / priority | GO proof required | NO-GO / next owner |
|---|---|---|
| F01 / P0 | union link coverage، missing/conflicting fail closed، competing return/invoice commit safety، stable IDs/history | false allow/protected deletion/orphan risk؛ Sales+MIS+Finance+Engineering |
| F06 / P1 | action-vs-view policy approved، explicit read-only/approver/rejector/poster negative fixtures، all entrypaths same authority | view grants rule/run/post writes unexpectedly؛ Finance+Product |
| F07 / P1 | boot/render/query call graph + tx/history delta tests prove business pure read under relevant engine flags/eligible data | render creates credit or uncertain writer call؛ Finance+Engineering; disable affected read exposure |
| HR runtime / P1 | real nonAdmin HR، current-profile/field/reviewer scope and per-effect reads/negative tests | Admin substitute/no actual role evidence؛ HR/environment owner |
| Finance runtime / P1 | nonAdmin Finance/read-only variants، cap/tab differences، evidence/report field guards | Admin substitute/unobserved enforcement؛ Finance/environment owner |
| Historical hierarchy / P1 | valid interval/event/attribution coverage、missing explicit، no current→past fabrication | assumed as-of/closed-period rewrite؛ HR+report owners |
| Posting idempotency / P1 | repeated/simultaneous/new-run same entitlement once، tx/key proof across engine and recipient change، tail accounting | find-before-insert only/unknown repost/duplicate tx/unprocessed hidden؛ Finance+Engineering |
| Refund / P1 | OPD-04 + source proof/authority/partial-full/reversal ledger reconciliation | flag=bankproof or cancel=refund؛ Finance+Product |
| Engine routing / P1 | OPD-09 approved map/active installed flags، one writer per entitlement/source/period، no retroactive key/rate rewrite | dual engine writes/shadow command/ambiguous routing؛ Finance+HR+Product |
| Export privacy / P1 | server-side row/field/scope/expiry and download negative tests with same dataset contract | visible link assumed safe/leaked bank/HR/customer fields؛ domain data owner |
| Unit/completion / P1 | OPD-06 unit/rounding/boundary + paid/remaining conservation | implicit conversion or borderline label guess؛ Finance+Product |
| Rollback and boot/schema / P1 | package/flags/old routes and compatible versioned no-unapproved-init-effect proof؛ in-flight effects reconcile | automatic seed/migrate/reset upon deployment or rollback； Engineering+environment owner |

OPD-08 maker/checker unresolved blocks mandated separation policy activation، not source-defined existing read inventory. Gate evidence unavailable keeps affected future command OFF؛ present plan remains ready.

## 19. Testing

Testing strategy future only؛ no plugin execution/tests/live mutation now. Disposable authorized staging/fixtures for write, race, provider and rollback tests؛ role accounts supplied explicitly، never silently create/change users as part of audit.

| Test class | Meaningful coverage / acceptance |
|---|---|
| Unit | proof classification/domain IDs/unknown linkage؛ namespaces/amount boundaries/report mode/result reconciliation using policy-fixed fixtures |
| Integration | repository+actual schema engines/legacy refs/multiple mirrors/hook graph؛ no orphan/duplicate/history overwrite; checked HR close+insert/access effects |
| Contract | old AJAX success/message/data + redirects/tab/shortcode attrs/tokens unchanged؛ new DTO/result schema and field allowlists |
| Negative permission | actual role×route/module/action/row/field including nonAdmin HR/Finance、read-only variants، inactive/reparented subjects、cross-scope rows、assigned Manager/current reviewer |
| Concurrency | same Case return vs invoice create، two reviewers/stale stage، hierarchy simultaneous changes، duplicate plan/submit/post؛ invariant before/after histories |
| Idempotency | same intent retry/new transport ID、新 run same entitlement、HR recipient change、partial/missing-tail retry、unknown lookup first |
| Migration compatibility | default release no schema/IDs/status/caps rewrite؛ future approved additive change backward package compatibility only if separately authorized؛ no migration execution merely for test label |
| Snapshot parity | old/new identical inputs with native-ID sets、same time/grain/scope/source/unit؛ expected semantics differences logged/explained and signed—not exact totals blindly |
| Role smoke | eight mission screens authorized scope/links/actions/states؛ Seller/Sup A/G/D baseline؛ HR/Finance real accounts mandatory for runtime acceptance |
| Browser QA | RTL/calendar/filter/page/deep-link/old-new routing、Loading/Empty/error/Partial/Unknown、allowed/disallowed action context؛ read purity observe effects in fixture, no redesign prescribed |
| Rollback/recovery | old package/views safe with post-cutover data؛ committed Finance tx remain، HR actual access/hierarchy preserved/reconciled؛ no blind replay |

DB read-only observability tests separate business tables/ledger/history from technical job/cache writes؛ still verify no unapproved DDL/cap/page seed. PHP lint/static checks useful future but not substitutes for enforcement/outcome integration tests. Test precise race/idempotency failures rather than mirroring implementation details. Performance limits derive from measured environment/source bounds؛ no invented throughput target.

## 20. Git / PR

**FUTURE WORKFLOW ONLY؛ GitHub setup later per user.** No git init/branch/commit/remote/push/PR/action taken. User's current workspace changes and accepted audit snapshot preserved.

Proposed branches `codex/<scope>`: e.g. `codex/foundation-query-contract`، `codex/financial-dependency-shadow`، `codex/seller-read-adapter`؛ names examples، not created. One PR = one adapter/contract or one guarded action family with fixtures and rollback؛ no UI rollout + capability rewrite + schema change in same PR. Immutable source inputs and deployment artifact baseline established in setup later، not force revert existing phase1 changes.

Required checks scoped: PHP syntax/compatibility، contract/negative scope، read-effect parity، applicable race/idempotency、old route/response regression، secrets/field leakage and no unapproved schema/cap delta. Sensitive financial/HR reviewer domain signoff + engineering review； policy-open feature cannot merge as active by only passing lint. PR description lead trigger/before-after behavior、evidence/limits/gate/flag/default/fallback، validated invariant refs.

Merge order logical: baseline contracts/tests→pure native adapters/shadow→authorized read models→bounded role exposure→individually gated write families. Explicit future approved additive schema PR separate if unavoidable، backward compatibility verified before dependent code activation. Release tags/artifact hashes map code+contract+schema compatibility+flags; prior rollback-compatible package retained. Financial execution records survive code revert. CI configuration itself future proposed، no workflow files created or GitHub actions executed.

## 21. Documentation

Proposed contents only؛ **these files not created/pushed**:

| Future file | Content / authority |
|---|---|
| README.md | plugin boot/prerequisites、artifact/version provenance、supported entry paths、local test setup、safe read/write distinction |
| AGENTS.md | scoped work rules، immutable business sources、no live/financial/access changes without task authorization، skills/tests/review/rollback expectations؛ no fabricated current repo rule |
| docs/ARCHITECTURE.md | layer/seam map、legacy bridge/hooks、domain/query/command boundaries、boot side effects، X/V1 references |
| docs/PRODUCT-INVARIANTS.md | G INV-001…036 binding with acceptance refs؛ OPDs not silently resolved |
| docs/ROLE-MATRIX.md | route/module/action/row/field boundaries、current vs target/cap evidence، real role runtime limits |
| docs/DESIGN-SYSTEM.md | approved Seller/Supervisor visual source refs and future integration scope؛ no business logic from prototype |
| docs/COMPONENTS.md | actual implemented shared/role-specific contracts when built؛ no claim conceptual needs already components |
| docs/QA-STRATEGY.md | test matrix/fixture privacy/races/role proof/read purity/rollback and gate evidence |
| docs/GITHUB_HANDOFF.md | later repo/branch/PR checks/artifacts/release tags/ownership/open gates؛ no secrets/access tokens |

Future ADRs can record transport/persistence/locking/flags decisions section23؛ copy frozen contracts by reference/version، no competing duplicate policy document. Documentation shall distinguish code verified/runtime verified/target assumption.

## 22. Readiness Matrix

| Foundation area | Planning readiness | Implementation activation status |
|---|---|---|
| Native query/role DTO/compatibility layers | READY, NO DB CHANGE baseline | future coding authorized separately؛ pure boot/query/scope tests before exposure |
| Case/source/ownership proof adapter | READY logical contract | unresolved OPD-01 lineage stays unknown； no persistent canonical IDs now |
| Shared F01 guard | READY contract | protected write rollout BLOCKED until union/race/history proof |
| Current role read UX integration | READY plan | phase-gated source/permission/purity؛ no UI design in this task |
| Metrics/report adapters/shadow | READY contract | F02/03/04 affected trusted metrics & F09 as-of CONDITIONAL |
| HR hierarchy/access read | READY with unknown/history states | role actual behavior NOT LIVE VERIFIED؛ current→past inference blocked |
| HR sensitive/final apply | READY wrappers/gates plan | BLOCKED relevant OPD-05/F08/HR authority/effect/runtime prerequisites |
| Finance read/review | READY conditional plan | real Finance runtime / F06/F07 / unit/evidence gates required |
| Posting/ledger/refund/engine | READY safety contract | BLOCKED OPD-04/06/08/09 + lock/key/coverage/unknown recovery proof |
| Maintenance/export/credential | READY boundary | action-specific policy/impact/privacy gates؛ no default broad activation |
| DB migration | NOT REQUIRED for baseline foundation | optional candidates NOT APPROVED؛ assess only when invariant need proven |
| GitHub/CI setup | FUTURE PLAN ONLY | deferred user-authorized setup؛ no action taken |

Ready plan does not certify existing unsafe path or guarantee zero-breakage without future checks. Safety architecture can be planned without destructive migration؛ some writes may require separately approved additive durability and remain OFF until then.

## 23. Open Implementation Decisions

Product policies remain canonical **X§19 / OPD-01…10**؛ engineering choices below do not reopen role design or invent duplicate policy. Existing gap refs supply owner/context؛ seam choices resolved before implementing affected path, not blocker to this plan.

| Engineering choice | Recommended default / unresolved evidence | Source dependency / owner |
|---|---|---|
| Authoritative deployed artifact vs accepted ZIP/workspace | verify hashes/build/constants/WP-PHP-DB actual environment read-only during future setup؛ no assume latest bytes | A/C provenance؛ environment owner + Engineering |
| New API transport/version envelope | keep current AJAX/admin-post compatibility; opt-in normalized adapter؛ choose AJAX vs REST after consumers/auth map | X permission/result؛ Engineering + domain owner |
| Adapter seam vs method extraction | wrap existing pure getters/query first؛ extract writer only if commit guard can't be safely placed؛ small PR | F01/F06/F07/HR-G08؛ Engineering |
| Atomicity/locking/intent persistence | assess actual DB engines/constraints/hooks and transaction coverage؛ do not promise atomic wrapper around legacy external effects | F01/FIN-IDEMPOTENCY/HR-G06/08؛ Engineering + Finance/HR/MIS |
| Durable dedup/correlation/alias storage | existing native refs/logs first؛ additive candidate only if invariant cannot be proven otherwise؛ migration not approved | OPD-01/09、FIN-LOCK/IDEMPOTENCY؛ Product + Engineering + domain owners |
| Flag persistence/ownership/configuration | logical per-role/screen/read/write gates؛ code/config/option storage decision later؛ not current financial/scope options repurposed | X compatibility/F06؛ Engineering + release owner |
| Boot/init purity isolation | list allowed technical initialization vs unapproved schema/cap/page business changes؛ preserve required legacy hooks under certified context | F07/G INV-026/027؛ Engineering + environment owner |
| Shadow query budget/cache invalidation | measure sources/cohorts/scale; scope/version keys and hierarchy invalidation؛ no arbitrary threshold | F02/03/04/F09、MIS-G06/10؛ Reporting + Engineering |
| External provider effects/retry | keep callback/token behavior؛ need official version-specific gateway/notification/refund contract when touched | OPD-04/09/FIN-RECOVERY؛ Finance + Engineering |
| Actual test accounts/fixtures | dedicated HR/Finance + negative variants; approved disposable staging only؛ no account create/change now | HR-G01/FIN-RUNTIME؛ environment owner + HR/Finance |
| Rollback recovery authority/durable effect evidence | reconcile committed refs first؛ actor/domain signoff، no automatic snapshot restore or entitlement debit | OPD-05/09、FIN-RECOVERY/HR-G08؛ domain owners + release owner |

No technical default decides bank proof、release policy、unit normalization、maker/checker、historical cohort or skipped-tier business rights. Safety-sensitive choice unresolved → affected action off/unknown diagnostic، not best-guess execution.

## 24. Freeze Review

| Review | Result |
|---|---|
| Transition without destructive migration | YES baseline adapter/read/guard contract؛ optional durability candidates explicitly unapproved |
| DB/IDs/history/routes/statuses/financial relations preserved | YES plan requires preservation + future regression/rollback proof |
| Current users/permissions unchanged | YES؛ future access delta explicit policy gate، no silent reset/remap |
| Read-first covers hidden side effects | YES includes F07/boot/shortlink/get-or-create/report technical writes |
| Write safety/unknown outcome shared | YES near-commit guards、per-effect proof、reconcile before retry؛ no dual financial shadow writer |
| Role phase order dependency-correct | YES F01/F07/scope/report foundations earlier، HR/Finance writes separately gated |
| OPD and runtime limits preserved | YES HR/Finance NOT LIVE VERIFIED؛ product policies not assumed |
| Implementation/migration/GitHub/CI begun | NO؛ plan and evidence documents only |

Validation document records ordered sections/local links/source-input preservation and 248 code baseline hashes. This is document/code-integrity checking، not live integration test or financial correctness certification. **STOP: no implementation starts after this deliverable.**

CRM IMPLEMENTATION FOUNDATION PLAN V1 READY
NO CODE CHANGED
NO MIGRATION PERFORMED
NO GITHUB ACTION TAKEN
