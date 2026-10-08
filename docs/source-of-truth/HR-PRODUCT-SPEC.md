# HR PRODUCT SPEC

تاریخ: 2026-10-06 · **FROZEN PRODUCT TARGET / IMPLEMENTATION DEFERRED** · **PRODUCT / WHAT–WHY ONLY**.

## 1. Executive Summary

HR مسئول **WORKFORCE + STRUCTURE + ACCESS LIFECYCLE + REQUEST APPLICATION + TEMPORAL HISTORY** است. تغییر وضعیت شغلی، hierarchy، access و compensation آثار متفاوت دارند و نباید با یک «ویرایش پروفایل» پنهان شوند. HR مالک اجرای مجاز تغییرات نیروی انسانی است؛ فروش درخواست/مرحله بررسی دارد، Finance مالک تصمیم مالی و ledger، MIS مالک source operations، و Admin مالک technical/system administration است.

**HR ROLE RUNTIME NOT LIVE VERIFIED.** حساب واقعی HR از مسیرهای موجود پیدا نشد؛ Admin #1 جای HR استفاده نمی‌شود. کل ادعاهای inventory، guard و workflow HR **CODE VERIFIED** هستند؛ role-specific UI behavior، enforcement و successful apply **NOT LIVE VERIFIED**. نبود حساب مانع Product Spec کدمحور نیست؛ readiness طراحی با readiness اجرای write متفاوت است.

اصل **KEEP + ADAPT** و تمام **Gate 0 INV-001…INV-036** binding. Structure target Freeze است؛ OPD-05 handover، credential/impersonation policy و temporal/compensation integrity activation dependencies آینده‌اند. هیچ code/plugin/DB/business data/permission تغییر نکرد، هیچ UI طراحی نشده.

مراجع:

- **A:** [Audit پذیرفته‌شده](CRM-PRODUCT-ROLE-ARCHITECTURE-AUDIT.fa.md)، بخش HR، F08 و shared risks.
- **G:** [Gate 0](GATE-0-PRODUCT-INVARIANTS.md)، HR/temporal/permission/history/compatibility و OPDهای باز.
- **C / S / D:** [Code Registry](CODE-EVIDENCE-AND-ACTION-REGISTRY.md)، [Status Catalog](STATUS-TRANSITIONS.fa.md)، [Decisions / Visibility](DECISIONS-VISIBILITY-INTERACTIONS.fa.md).
- Role boundaries: [Senior](SENIOR-SUPERVISOR-PRODUCT-SPEC.md)، [Manager](SALES-MANAGER-PRODUCT-SPEC.md)، [Deputy](SALES-DEPUTY-PRODUCT-SPEC.md)، [MIS](MIS-PRODUCT-SPEC.md)؛ Seller/Supervisor از A/G/D. Prototypes فقط visual references هستند.
- **E:** [بررسی دسترسی و کد HR این مرحله](../evidence/hr-access-and-code-review-2026-10-06.md)؛ [Role Access Map](ROLE-ACCESS-MAP.fa.md).

Evidence: LIVE VERIFIED فقط discovery با actor Admin در E؛ CODE VERIFIED وجود code branch؛ BOTH فقط اگر همان ادعا از runtime واقعی HR و code توافق کند، **اکنون هیچ workflow HR این برچسب را نمی‌گیرد**؛ INFERENCE target contract؛ NOT VERIFIED نبود proof. Confidence static inventory/guards HIGH، policy completeness MEDIUM، runtime UNKNOWN.

## 2. Role Mission & Boundaries

| قرارداد | WHAT / WHY |
|---|---|
| Role Mission | نگهداری workforce و structure/access lifecycle قابل ردیابی، اعمال مجاز درخواست و حفظ temporal history |
| Daily Jobs | consistency نیرو/manager/access، pending/failed requests و current reviewer، exceptions و actual applied outcome |
| Occasional Jobs | onboarding، transfer/termination، position/level/unit changes، compensation periods، import/export مجاز؛ credentials فقط مسیر restricted مستقل |
| Decision Responsibilities | تشخیص هویت/تطبیق profile-user؛ تغییر workforce مجاز؛ final HR apply با impact/handover؛ حفظ تاریخ اثر و authority boundaries |
| Data Needed | person/profile/user IDs، employment/position/manager، access linkage، effective dates/history، request chain، reason/result، compensation طبق field permission |
| Data Not Needed | sales/customer details بی‌ارتباط، invoice operation/Finance posting، bank data غیرضروری، plaintext credential، raw MIS repair |
| Allowed Responsibilities | workforce/structure/request/compensation code-supported domains با target granular guards؛ scope/field/action separation |
| Explicit Non-Responsibilities | Sales operator، Finance approval/refund/posting، MIS، unrestricted WP Admin، historical sales/credit owner rewrite |

Mission **INFERENCE / HIGH** مبتنی بر A/C؛ job frequency target است. HR سازمان را می‌بیند ولی role-management gate broad کنونی policy کامل تمام sensitive actions نیست. «قطع همکاری» به معنی business handover کامل نیست.

## 3. Current Verified Inventory

Inventory زیر از A/C است؛ شمارش/ظاهر Admin دلیل HR runtime نیست. E workforce six profiles و zero hidden را فقط برای access discovery دید؛ هیچ account HR نبود. Frontend `/crm-hr/` با shortcode `sn_hr_panel`، HR position route resolver؛ WP admin menus عموماً `manage_options` و مسیر frontend HR gate مستقل.

| Section / feature | CODE VERIFIED current | Classification | Decision / rationale |
|---|---|---|---|
| `hr-workforce` | user/profile/current parent/position/level/employment، search/filter/inline update | ESSENTIAL | KEEP؛ identity domain و sensitive deltas روشن |
| Readiness/audit cards | missing parent/level/compensation، visible/hidden counts | ESSENTIAL | KEEP context-sensitive؛ no-parent همیشه defect نیست |
| Inline autosave | برخی position/manager/status/type fields | CONFUSING | SIMPLIFY / separate sensitive change intent؛ save success از appearance فرض نشود |
| Manual invoice / extra-number overrides | allow/deny/inherit، parent inheritance rules | ESSENTIAL | KEEP؛ effective access inspector/context، no capability redesign now |
| `hr-manual-add` | matching/creating WP user + HR profile، position/manager | ESSENTIAL | KEEP onboarding؛ identity conflict و technical credentials مستقل |
| `hr-csv` import/export | CSV/XLSX simple/advanced/roundtrip، dry-run/APPLY | SUPPORTING | KEEP، scope/provided fields/outcomes honest |
| `hr-bulk` | selected/filter-based profile updates | SUPPORTING | SIMPLIFY exact server row/field scope؛ sensitive changes conditional |
| `hr-change-requests` | transfer/terminate request chain و HR final apply | ESSENTIAL | KEEP؛ step approval≠applied |
| `hr-compensation` | current profile و temporal salary/commission settings/history | ESSENTIAL | KEEP separate compensation domain؛ payroll/interval integrity conditional |
| `hr-positions` positions/levels | CRUD/mapping/defaults/active | SUPPORTING | MOVE Structure & Access؛ dependency checks before destructive action |
| `hr-extra` legacy profile editor | مسیر موازی profile action | REDUNDANT presentation | MERGE editor context؛ compatibility fields/history preserved |
| Missing-profile users / legacy mappings | diagnostics و onboarding repair context | SUPPORTING | MOVE Exceptions/advanced access؛ no implicit repair-all |
| `hr-overview` | profile counts by role/level/readiness | SUPPORTING | MERGE actionable Workforce overview؛ count≠performance score |
| `hr-structure` | department/unit/team links | SUPPORTING | KEEP structure entity جدا از reports_to hierarchy |
| `hr-hierarchy` | assign/move/replace، cycle/history | ESSENTIAL | KEEP؛ temporal/impact contract |
| `hr-logs` | old/new/context و last-operation report | ESSENTIAL | MERGE workforce timeline، raw technical data advanced |
| Staff panel viewing / return | broad HR gate، session switch، expiry | SUPPORTING / sensitive | MOVE restricted support policy؛ not universal readonly |
| Password/reset/SMS/bulk password | credential handlers under HR gate | MISPLACED in ordinary CRUD | MOVE credentials policy، independent sensitive authority target |
| Document/training workflow | columns contract/training status؛ complete workflow not established | MISSING / NEEDS VALIDATION | no invented document/training lifecycle |
| Termination handover impact | complete flow not established | MISSING | ADD target prerequisite under OPD-05؛ implementation deferred |

Code locator: [class-sn-plugin.php](../source/sales-network-v4/includes/class-sn-plugin.php) `sn_can_manage_hr_panel:9112`، renderer `10586`، profile/inline/bulk `18451/18669/18744`، request `10397–10439`؛ [HR transfer](../source/sales-network-v4/includes/class-sn-hr-transfer.php) search/export؛ [Hierarchy Service](../source/sales-network-v4/includes/class-sn-hierarchy-service.php) assign_parent `29`.

Required verification account: **non-Admin `sn_hr` + base `read` + active HR profile position slug `hr`**؛ route `/crm-hr/`. `sn_manage_hr` capability جدید/لازم اختراع نمی‌شود. Frontend gate role یا position می‌پذیرد؛ final-request/helper gates position-dependent‌اند، بنابراین role/profile consistency باید با account واقعی تست شود. هیچ account ساخته/تغییر نشده.

## 4. HR vs Sales / Admin

### HR vs Sales Management

| Capability | Sales Management | HR | Why Different? |
|---|---|---|---|
| Workforce profile | scoped context و درخواست | create/edit workforce مجاز | sales view≠HR profile mutation |
| Hierarchy | current team visibility | current hierarchy change و temporal records | scope today از attribution گذشته جدا |
| Transfer request | submit/review assigned step | authorized final application | intermediate approval تغییر اعمال‌شده نیست |
| Termination | request/operational handover input | employment/access change با approved impact policy | disable profile≠complete handover |
| Access | actual permissions برای کار نقش | business access lifecycle تحت guard | position، role و credential مستقل |
| Current manager | responsible team context | effective parent assignment | تغییر parent credit تاریخی نمی‌سازد |
| Historical attribution | event/Case/commission reference | preserve، no rewrite | HR تاریخ past sales را مالک نیست |
| Active/inactive | recipient eligibility/current visibility | workforce lifecycle state | inactive لزوماً full auth revocation نیست |
| Impersonation | no inherited HR support right | code-supported sensitive support، policy conditional | not ordinary workforce edit یا readonly sandbox |
| Compensation | own entitlement مجاز، not staff salary admin | compensation domain code-supported | no Finance posting/engine ownership |
| Operational ownership | current sales work/next action | impact/handover coordination | HR actor Seller یا arbitrary reassignment owner نیست |

### HR vs technical/system Admin

| Concern | HR contract | Admin boundary |
|---|---|---|
| User account creation | onboarding linkage code may create user؛ internal business scope | arbitrary WP/system users و privileges از HR نتیجه نمی‌شود |
| Role/capability | plugin business-position mapping و explicit permission impact | unrestricted capability/role admin خارج target |
| Password/credentials | current code accepts HR gate؛ separate restricted policy | system admin credential powers خودکار اعطا نشوند |
| Impersonation | current handler HR gate، non-Admin internal target؛ policy separate | Admin availability proof HR runtime نیست |
| Plugin/settings | not ordinary HR mission | technical installation/settings Admin-only where code requires |
| Workforce data | HR authority مجاز | Admin broad access مسیر جدا، minimum necessary در HR |
| Business hierarchy | HR effective structure با history | technical admin مجوز rewrite past sales ندارد |

Current code HR برخی technical-sensitive actions دارد؛ آنها را **admin-only خلاف evidence** اعلام نکنید. Target separation مجوز فعلی را تغییر نمی‌دهد؛ restricted placement/activation policy و QA لازم است. HR نباید unrestricted WordPress Admin شود.

## 5. Target IA

| Module | Placement | WHAT / WHY / decision |
|---|---|---|
| Workforce | PRIMARY | canonical workforce context و actionable readiness؛ KEEP / consolidate parallel editors |
| Onboarding / Transfer | PRIMARY | identity linkage و changes with effective impact؛ KEEP stages مستقل |
| Requests | PRIMARY | pending/current reviewer/final apply/failed و history؛ KEEP |
| Structure & Access | PRIMARY | position/level/unit/hierarchy + effective business access؛ KEEP، role/capability distinct |
| Compensation | CONDITIONAL PRIMARY for authorized HR | sensitive current/period/history؛ KEEP separate، field permission |
| Exceptions / Audit | PRIMARY | inconsistency، failed apply و temporal gaps؛ target queue + existing logs |
| Credentials / Impersonation | ADVANCED / RESTRICTED، policy CONDITIONAL | separate support authority/context، not ordinary workforce step |
| Import / Bulk / Export | SECONDARY / CONDITIONAL | large scoped changes و governed field contract؛ no batch destructive default |
| Legacy mapping / raw diagnostics | ADVANCED | compatibility troubleshooting؛ history/values intact |

IA **INFERENCE / HIGH**؛ current implementations CODE VERIFIED. target consolidation = presentation/update contract مشترک، نه destructive profile/user merge یا DB redesign. هیچ sidebar/layout/component بصری ساخته نشده.

## 6. Workforce Model

| Entity / field | Product meaning / preserved identity |
|---|---|
| Person | انسان؛ legal/business identity context؛ global person resolver implementation کامل فرض نشود |
| Workforce Profile | HR record ID و employment lifecycle؛ با WordPress ID یکی نیست |
| WordPress User | technical account ID/login/access linkage؛ one-to-one constraints باید از current proof، نه حدس |
| Role | WP permission container / legacy role؛ position نیست |
| Position / level | سمت و طبقه business؛ mappingها influence access دارند ولی permission object مستقل است |
| Hierarchy Assignment | reports_to child/parent و effective interval/history؛ user/meta alone canonical hierarchy نیست |
| Unit/department/team | organizational structure، لزوماً همان reporting parent نیست |
| Employment/access state | active/inactive/terminated… با actual auth/session/product access جدا |
| Effective dates / history | declared business effective time، applied time و original event time مستقل |
| Contact/business fields | necessary identity/contact data، sensitive subsets و field guard |

Canonical HR profile target: stable profile/person/user linkage، employment status/type، position/level، current parent و interval، activity/access state، contact/business fields، role/capability linkage و history. این logical model **INFERENCE** است، نه schema جدید.

Code workforce merges profile/user context و optional overrides؛ missing linkage/duplicate profile conflict، نه create/merge خودکار. Search loaded list با global user search متفاوت؛ row bounds/freshness اعلام شوند. no parent/no level/no compensation readiness باید role-dependent باشد؛ head-of-tree بدون parent الزاماً defective نیست. staff count KPI عملکرد HR، readiness percentage یا target نیست.

## 7. Temporal Hierarchy

G INV-028/029 binding: current changes نباید historical sales owner، Event Actor، Credit Owner یا past hierarchy attribution را rewrite کنند. Missing history **UNKNOWN**؛ current manager fallback تاریخ گذشته نیست.

Current C `SN_Hierarchy_Service::assign_parent:29`: existence/self-parent/cycle checks؛ closes current assignment، new reports_to با **application-time** effective_from و history؛ end current sets effective_to. No explicit scheduled/backdated transfer proof در این helper. closing+insert atomicity و insert failure recovery کامل از success return code ثابت نمی‌شود.

| Target context | Requirement |
|---|---|
| Current hierarchy | validated current parent/position و active constraint policy؛ not credit attribution |
| effective_from / effective_to | valid interval، boundary/timezone مشخص؛ intended business effect versus application time جدا |
| Historical manager / position | snapshot/event/interval معتبر؛ historical position completeness code paths بررسی‌نشده، UNKNOWN if missing |
| Transfer event | actor/subject/from/to/reason/request/time/result؛ immutable reference |
| Original attribution | Sales/source/invoice actor و original owner حفظ |
| Credit boundary | past entitlement snapshot جدا؛ compensation engine/Finance مالکان مستقل |

Future change preview باید old/new parent و affected scope/access/dependency نشان دهد؛ preview grant final write نیست. No silent backfill of historical managers، enforced scheduled engine یا forced migration. Temporal integrity gap **PRODUCT GAP — DO NOT IMPLEMENT**؛ UI می‌تواند timeline honest با missing intervals داشته باشد.

## 8. Onboarding

Current manual-add/profile/import handlers code-supported؛ workflow کامل legal documents/training اثبات نشده. **CODE VERIFIED lifecycle pieces / NOT LIVE VERIFIED runtime**.

| Stage | WHAT / WHY |
|---|---|
| Identity discovery | match existing person/profile/WP user با stable refs؛ ambiguous mobile/login/name conflict قبل apply؛ no accidental duplicate |
| Profile creation/linkage | existing user link یا code-authorized internal creation؛ user/profile IDs مستقل و existing history preserved |
| Position/level placement | business classification؛ access mapping impact explicit |
| Hierarchy placement | current validated parent، no cycle، reason/effect context؛ position-sensitive parent policy |
| Access activation | product permissions و auth account effects distinct؛ active flag تنها proof usable login نیست |
| Required data | fields actually required by supported handler؛ contract/training status columns≠document workflow |
| Effective/completion state | profile created، account linked، access applied، hierarchy applied و unresolved effects جدا؛ one success banner completion proof نیست |

Credentials issuance/reset/SMS بخش13، نه mandatory ordinary onboarding step؛ no new training/checklist/document-upload/approval pipeline. Import CSV/XLSX roundtrip `sn_hr_users_v1`، provided fields/identity matching preserved؛ dry-run report may update options و purely read نیست. APPLY nonce و conflict checks CODE VERIFIED، safe overwrite/atomicity runtime NOT VERIFIED. 5MB/10,000-row UI limits accepted، scale guarantee نیست.

## 9. Transfer

Current request type `transfer_seller`، Sales requester/subject scope/reason، from-parent snapshot و target-parent، reviewer chain؛ HR final `assign_parent` در C. Generic every-position transfer request نیست؛ direct hierarchy edit مسیر مستقل code موجود دارد و نباید با request approve یکی شود.

Target record: requester/source، subject profile/user، from/to manager/position context، reason، reviewer/current/next، requested effective date **اگر supported**، actual effective/applied time و outcome/history. Current helper uses apply-now timestamps؛ scheduled date promise **NEEDS VALIDATION**.

Target steps: read fresh request/current hierarchy → authorized review → handover/impact policy under OPD-05 → final apply → actual before/after proof → approved/applied or failed/incomplete. Intermediate Sales approval≠applied transfer. Current HR may finalize pending_review too؛ enforce-complete Sales chain universally خلاف code proof است، bypass policy context نیاز دارد.

Preserve historical attribution، existing Case ownership rules until explicit handoff، current custody policy و credit history. Changing hierarchy visibility لزوماً Case assignment یا financial entitlement را منتقل نمی‌کند. inactive target، changed subject/current parent و already applied/conflicting request recheck لازم؛ current runtime/state-race enforcement NOT VERIFIED.

**OPD-05 DEFERRED / CONDITIONAL:** successor/open work/finance/tasks/custody handover و effective timing unresolved؛ no automatic reassignment، invoice relink یا commission rewrite. Future QA/rollback لازم؛ current transfer code path حذف نشده.

## 10. Termination

Current `terminate_seller` apply updates profile `is_active=0`، employment_status terminated، termination_date Tehran today و user meta sn_is_active0؛ log+success branch. Role removal، session revocation و complete owner transfer از این branch نتیجه نمی‌شود؛ write-result verification/partial failure نیز کامل اثبات نشده.

| Concern | Target contract |
|---|---|
| Request/review | supported subject/type و current reviewer chain؛ final HR authority separate |
| Effective date | intended effect versus actual apply date/time؛ scheduled removal policy NEEDS VALIDATION |
| Active/access | employment inactive، product access، WP role/auth/session state مستقل؛ access removal timing explicit policy |
| Open work/current custody | list protected Case/task responsibility و named valid next actor under OPD-05؛ ownerless work مجاز نیست |
| Future tasks | eligibility inactive/assignment policies clarified؛ no new scheduler یا guessed cancellation |
| Invoice/customer responsibility | responsible Sales/Finance actor و protected links retained؛ termination≠financial deletion |
| Compensation/entitlement | earned historical credit preserved؛ future entitlement/effective periods separate decision، no automatic clawback |
| History | actor/subject/request/reason/effective/applied before-after/outcome محفوظ |

**Disable user/profile ≠ complete business handover.** هدف impact/handover شرط architecture است؛ recipient policy هنوز **DEFERRED / CONDITIONAL**، نه recipient default Manager. Existing termination code unchanged؛ UI نباید completed offboarding ادعا کند وقتی فقط employment flag changed. Failed/incomplete apply باید evidence باقی و recovery بدون blind repeat داشته باشد.

## 11. HR Requests

Lifecycle labels product concepts‌اند؛ existing internal status IDs S حفظ. Requested event→`pending_review` یا `pending_hr`؛ intermediate approve/reject؛ HR apply→`approved`+applied_at یا `failed`؛ reject→`rejected`. «apply» یک operation است، status جدید اجباری نیست.

| State / transition | Current code / target interpretation |
|---|---|
| Requested | requester/subject/type/reason/from-target؛ pending duplicate guard |
| pending_review | non-HR فقط current_reviewer_user_id خود؛ HR reviewer code می‌تواند review کند |
| Intermediate approved_step | next parent reviewer یا pending_hr؛ completed personnel change نیست |
| Rejected | reason/notes و approval_path؛ current reviewer cleared؛ historical request باقی |
| pending_hr | HR final authority؛ no Sales final application |
| Final apply | transfer hierarchy یا termination fields؛ actual effects independently verified target |
| approved/applied | code records approved + applied_at after apply success؛ target requires proof actual change، not just branch banner |
| failed | failure reason/path و unresolved effects؛ failed status current reviewer/pending retries خودکار مجوز نمی‌دهد |

Must show requested change versus actual applied change، subject/from/to، current/next reviewer، reason/history، effect/applied dates. HR requests list کد برای HR broader all درخواست‌ها bounded، Sales requester/current reviewer filter؛ scope broad read final authority policy همه fields نیست. Current HR-context renderer create form نمی‌سازد، هرچند shared create guard HR position را می‌پذیرد؛ submit entry **CONDITIONAL / NEEDS VALIDATION**، نه invented always-visible form.

Fresh row/state/subject/target checks و race/duplicate apply safeguards target‌اند؛ code not universal transaction/idempotency proof. Recovery policy برای failed apply باید definitive effects را reconcile کند؛ approve again blind ممنوع. **CODE VERIFIED transitions / NOT LIVE VERIFIED successful workflow**.

## 12. Structure & Access

Business Position، Product Permission، WP Role/Capability و Credential چهار object/contract متفاوت. Unit structure، reports_to assignment و level classification نیز جدا. View≠Write تحت G INV-023…025.

Current `sn_hr_apply_user_access_for_position:9158` سمت را به operational role/caps plugin map می‌کند و previous plugin-managed roles را کنار می‌گذارد؛ unrelated WP roles/caps/history طبق مسیر code intentional preserved. اثر indirect workforce edit روی role/access واقعی است؛ no new mapping design یا automatic reset پیشنهاد نمی‌شود.

| HR responsibility | Target contract |
|---|---|
| Positions/levels/structure | CRUD با dependency refs و role mapping impact؛ delete/deactivate مستقل از label edit |
| Current hierarchy | effective intervals، cycle/legal-parent/active policy و impact scope؛ not historical sales attribution |
| Employment activity | access effects و eligibility state explicit، no assumed session disable |
| Business access profile | actual source of permission: position/role/direct override/inherit، effective result و reason |
| Manual invoice / extra-number overrides | current allow/deny/inherit precedence preserved؛ per-user versus parent effect روشن |
| Capability visibility | read inspector conceptual؛ unrestricted WP capability editor خارج role target |

Mapping mismatch/legacy role coexistence diagnosis≠forced remap. Inline/bulk/profile/manual/import paths باید intent/fields consistent داشته باشند؛ target consolidate update contract، هیچ code/service جدید اجرا نشده. Sensitive access delta باید reason/review/impact و field restriction داشته باشد؛ ordinary contact edit آن را پنهان نکند.

## 13. Credentials / Impersonation

**Separate restricted policy**؛ current CODE VERIFIED HR gate این actions را می‌پذیرد، پس admin-only بودن خلاف evidence است. Desired granular activation policy **DEFERRED**؛ existing permission change انجام نشده.

| Action | Current code evidence | Target boundary |
|---|---|---|
| Create/link account | manual onboarding/import may create/link technical account | internal intended identity؛ privilege creation/credentials جدا، no arbitrary WP Admin |
| Single password reset | `19060`: HR gate+nonce، existing user، password/repeat validation؛ reset then optional SMS، audit no password | target eligible internal nonprivileged subject restriction باید مستقل proof/policy؛ inspected handler such target guard نشان نمی‌دهد؛ CODE gap، runtime bypass NOT VERIFIED |
| Bulk reset | HR gate+nonce+APPLY، selected/all_active_hr/all_hr profile IDs، per-user results | technical-sensitive conditional، نه معمولی Workforce bulk؛ no blanket same-password recommendation |
| Credential send | reset و SMS separate outcomes؛ notification mirror sensitive flag | ارسال explicit action authority، secret storage/log policy و recipient proof؛ SMS failed≠reset failed |
| Staff panel view | `2287`: HR gate، no nested view، self/Admin/internal target guard و resolved route، actor-target transient30min/auth switch | support purpose only؛ code 30min transient ضمانت automatic logout/revocation نیست؛ not readonly sandbox |
| Return | nonce/transient actor restore `2312` | actor/target context و return مسیر؛ expire/recovery policy needs QA |

Purpose target: authorized support/verification، نه perform sales/finance action به نام employee. Current impersonation به target authentication تبدیل می‌شود؛ blanket write blockade اثبات نشده. Sensitive customer/compensation/financial data از target scope ممکن است expose شود؛ exact field/action restrictions **NEEDS VALIDATION**، نه «همه readonly».

Password reset audit logs reset/send/masked phone و password_logged=false؛ این عدم ذخیره secret در تمام SMS/mirror/notification backend را ثابت نمی‌کند. Durable start/end/reason impersonation audit در inspected handler اثبات نشده؛ transient actor-target audit trail immutable نیست. Target audit: actor/target/purpose/start/end/outcome و restricted fields/actions، no credential secret.

هیچ credential/reset/SMS/impersonation در این مرحله اجرا نشد. No policy تازه امنیتی را implementation-complete اعلام کنید؛ separate guard و QA/rollback future شرط activation-sensitive است.

## 14. Compensation

Current code HR compensation profile/history reads و edits زیر HR gate دارد: base_salary، IRT/IRR currency، commission_enabled/rule reference/override mode، effective dates، employment/position/level snapshots و reason/history. **CODE VERIFIED authority paths؛ role runtime NOT VERIFIED**. Commercial policy دقیق granular read/edit needs validation.

| Domain | HR responsibility / boundary |
|---|---|
| Compensation see | authorized employee current/period compensation و history؛ not all-row exposure default |
| Compensation edit | validated effective interval، unit/reason و allowed fields؛ backdated/locked correction policy conditional |
| Commission relation | per-person eligibility/override/rule reference input؛ rule-engine design/Finance posting مالک HR نیست |
| Effective history | before/after and snapshots محفوظ؛ no earned-credit retroactive rewrite |
| Salary versus Wallet | salary/compensation record≠wallet transaction/commission payout یا Sales performance |
| Finance | payroll/ledger/post/refund decisions مستقل؛ HR field change payment proof نیست |

C `sn_hr_compensation_update:11231` current profile و `handle_hr_compensation_timeline_action:19614` temporal path متفاوت‌اند. Timeline APPLY/date/reason checks دارد ولی previous interval را پیش از remaining overlap/insert می‌بندد؛ encompassing transaction در handler دیده نشد. Failure after close و locked_after_payroll enforcement بین paths **NEEDS VALIDATION**؛ full payroll lock از column یا helper خاص نتیجه نشود.

Target interval integrity: valid nonoverlap، effect/applied time distinct، protected paid-period guard و rollback بدون history loss. Currency conversion/rounding و entitlement routing تحت G OPD-06/09؛ تغییر snapshot امروز past sales credit را بازنویسی نکند. No payroll workflow invented، no wallet/Finance engine merge. salary masking in audit formatter protection تمام endpoints/export نیست.

## 15. Exceptions / Audit

Underlying diagnostics/code-supported states؛ unified queue target **INFERENCE**. No SLA یا «pending too long» threshold جدید؛ age فقط از timestamp معتبر و severity بر اساس state/impact، نه guessed timer.

| Exception / owner | HR responsibility / allowed action | Restricted action / resolution proof |
|---|---|---|
| Invalid hierarchy؛ HR structure | read cycle/self-parent/parent state، authorized corrected current assignment | no past hierarchy rewrite؛ valid interval+history و current scope proof |
| Inactive manager with active staff؛ HR/Sales owners | inspect impacted branches و allowed structure proposal | no automatic successor/custody transfer؛ confirmed current parent+business handover under OPD-05 |
| Pending request / age؛ current reviewer | view current/next step، review if authorized | no SLA/escalation command؛ approved_step or final applied proof distinct |
| Failed apply؛ HR/action service | inspect actual effects/history، approved recovery policy | no blind reapply؛ definitive before/after و reconciled state |
| Duplicate workforce profile؛ identity owner | compare native refs/mapping، explicit repair conditional | no destructive user/profile merge؛ stable linkage evidence/history |
| Missing user linkage؛ HR/technical account owner | onboarding identity match/create only code-authorized | credentials/privileged user creation restricted؛ linked profile/user proof |
| Role-position mismatch؛ access owner | effective access inspect و intended mapping delta | forced full reset/role remap ممنوع؛ allowed exact mapping/history |
| Termination with open work؛ HR+Sales/Finance | impact list/next actor and handover policy | no ownerless Cases/financial deletion؛ named responsibility/eligible outcome proof |
| Out-of-scope request؛ request owner | deny/conflict و scope reason | ancestor rank bypass ممنوع؛ authorized subject/reviewer proof |
| Access mismatch/apply failed؛ access service | intended versus actual permissions compare | arbitrary WP Admin changes؛ exact applied/partial failures and audit |
| Historical attribution gap؛ history/domain owners | UNKNOWN و limited interpretation | infer past from current manager ممنوع؛ valid timestamp/snapshot evidence |

Audit required: **actor، subject/profile/user/request، requested change، before/after، reason، reviewer chain، effective date، applied_at، result/failure reason، correlation/reference**. Logical context، نه DB schema جدید. Sensitive secrets و unnecessary salary/bank/PII در audit raw payload exposure default نباشند.

Current profile logs/hierarchy history/approval_path و compensation history CODE VERIFIED؛ immutable complete audit/all writes rollback NOT VERIFIED. overwritten last-operation report historical log کامل نیست. Target request/result links preserved؛ technical JSON advanced با field permissions.

## 16. Permissions / Bulk Actions

Conceptual product permissions؛ WP capability names جدید نیستند. Common read guards: authenticated appropriate module، subject/row scope، sensitive field visibility و query bounds/freshness. Current HR management gate broad است؛ target separation مجوز فعلی را silently reset نمی‌کند.

| Concept | Assessment |
|---|---|
| `view_workforce` | KEEP scoped organization workforce context؛ runtime NOT VERIFIED |
| `create_workforce_profile` / `edit_workforce_profile` | supported code؛ target identity/field guards و sensitive delta separation |
| `view_hierarchy` / `edit_current_hierarchy` | read/history versus current assignment separate؛ temporal/impact policy |
| `submit_hr_request` | shared code supports HR، current HR create entry conditional |
| `review_hr_request` / `apply_hr_change` | step/reject/final effects separate؛ guard/state/recovery |
| `view_access_state` / `manage_business_access` | permission source inspect versus mapped change؛ no WP Admin superset |
| `manage_credentials` / `impersonate_user` | code-supported broad HR gate، target ADVANCED/RESTRICTED conditional policy |
| `view_compensation` / `edit_compensation` | separate sensitive read/write field scope و temporal/payroll constraints |
| `view_hr_audit` / `export_hr_data` | context/history read versus export field/scope explicit؛ no secret output |

Sensitive write target guards؛ successful current enforcement ادعا نمی‌شود:

| Action | Scope / state guard | Row guard / field restriction | Audit |
|---|---|---|---|
| Create/edit profile | eligible internal subject/intended lifecycle، identity unambiguous | profile/user mapping fresh؛ ordinary fields جدا از access/compensation | actual before/after، actor/reason/result |
| Current hierarchy/transfer | permitted HR subject/parent، cycle/active/effective policy | current parent/request/interval unchanged؛ no historical attribution fields | from/to، requested/applied effect و history |
| Termination/apply request | HR final authority، supported pending state/impact policy | subject/current request/recipient work validated؛ no invoice/credit rewrite | chain، open-work disposition، result/failure/partial proof |
| Business access change | allowed position/override mapping، policy effect time | target role/state/profile fresh؛ no unrelated capabilities reset | exact permission delta/source و applied outcome |
| Credential reset/send | independent restricted authority/eligible target policy | internal/privileged target boundary validated؛ only intended reset/send، no secret logs | reset versus send result، masked contact، actor/reference |
| Impersonation | purpose/eligible target/no nested/expiry policy | target route/scope fresh؛ restricted actions/data policy explicit | actor/target/start/end/purpose/outcome؛ durable trail requirement |
| Compensation | sensitive permission، date/unit/locked paid-period policy | exact profile/user/current periods؛ salary/rule-reference only، no Finance posting | before/after period، reason/effect/applied/result |
| Export/import/bulk | allowed dataset/selection/filter و field contract، APPLY for writes | exact subject IDs/server scope، provided fields، conflict checks | run/source/rows/fields/per-item outcomes |

| Bulk action | Classification | Rationale |
|---|---|---|
| Bulk Workforce Update | SUPPORTED code / CONDITIONAL field scope | exact selected/visible/filter target IDs و provided fields؛ sensitive deltas مستقل |
| Bulk Transfer | SUPPORTED hierarchy tooling / CONDITIONAL | direct hierarchy bulk≠batch request finalization؛ OPD-05 and temporal guards |
| Bulk Termination | NOT ALLOWED در frozen baseline | batch offboarding/owner handover proof ندارد؛ code broad update possibilities policy مجوز عمومی نمی‌سازند |
| Bulk Access Change | CONDITIONAL / NEEDS VALIDATION | position update می‌تواند access change دهد؛ granular review/fields/race needed |
| Bulk Credential Reset | SUPPORTED code / RESTRICTED CONDITIONAL | existing technical-sensitive handler؛ ordinary HR bulk نیست، no blanket batch reset default |
| Bulk Compensation Update | SUPPORTED optional profile/import pathways / CONDITIONAL | explicit fields/period guards و financial sensitivity؛ timeline integrity QA |
| Bulk Export | SUPPORTED code / CONDITIONAL | selected/all distinction، sensitive whitelist/row scope؛ runtime NOT VERIFIED |

Default conservative؛ REQUIRED destructive batch تعریف نشده. Target NOT ALLOWED capability فعلی را حذف نمی‌کند. Truthful per-item applied/skipped/failed/unknown و counts INV-034؛ one success summary≠all completed. Credential reset success+SMS failure و HR flags success+handover incomplete نتایج جدا.

## 17. Data Visibility / States

Minimum necessary؛ HR sensitive است. Scope organization-level code gate به معنی unlimited PII/compensation/export exposure نیست.

| Visibility | Contract |
|---|---|
| MUST SEE | profile/user identity refs، employment/position/parent/activity، effect/history، intended/actual access، reviewer/request/result/reason؛ compensation فقط authorized domain |
| NICE TO HAVE | allowed readiness filters، structure/history diagnostics، aggregate issue counts و impact summaries، field-masked audit context |
| SHOULD HIDE / RESTRICT | plaintext passwords/secrets/tokens؛ unnecessary bank/personal sensitive fields؛ compensation خارج permission؛ raw sensitive audit details؛ unrelated other-user/customer records؛ Finance/MIS/Admin controls |

Reports/export: CSV roundtrip `sn_hr_users_v1`، selected/all و streaming paths موجود؛ selected cap5000 و chunks500 در accepted C؛ screen filter≠all export. explicit permission، server IDs/current scope و sensitive field whitelist در output لازم؛ salary log masking تضمین whole export نیست. Import dry-run preview metadata writes، not business apply؛ runtime overwrite/security proof pending.

| State | HR meaning / example |
|---|---|
| Loading | workforce/request/history read pending؛ not zero counts |
| Empty | authorized complete scope without record/request؛ failed service≠empty |
| No Result | filter no match؛ no whole-org claim |
| Unauthorized | action/subject/field denied؛ no fake empty masking |
| Stale | profile/manager/access/report cached؛ freshness explicit |
| Incomplete | linkage/history/compensation partial، termination handover incomplete |
| Conflict | request changed before review، duplicate profile، inactive target، interval/parent conflict |
| Partial Success | profile/hierarchy/access effects متفاوت یا bulk rows partial؛ completion claims honest |
| Retryable Failure | known read/apply failure with authorized safe recovery، not blind repeat |
| Outcome Unknown | write response lost؛ actual profile/access/history reconcile first |
| Offline | connection unavailable؛ cached state stale، new change applied فرض نشود |

Effective-date conflict workflow problem است؛ current employment active field historical access proof نیست. historical attribution unavailable UNKNOWN؛ target manager inactive denial requires fresh policy check. Credential/send outcomes و approval/application outcomes namespaces مستقل، internal status IDs محفوظ.

## 18. Product Gaps / Deferred

| ID / Priority | Status / Evidence / confidence | Limit / condition |
|---|---|---|
| HR-G01 / P1 — runtime access | DEFERRED / CONDITIONAL؛ E/A | HR ROLE RUNTIME NOT LIVE VERIFIED؛ non-Admin sn_hr/read + active profile hr لازم؛ design blocker نیست |
| HR-G02 / P0 — termination/transfer handover | PRODUCT GAP — DO NOT IMPLEMENT؛ G/OPD-05 | open work/custody/finance/next actor، no ownerless outcome؛ recipient policy unresolved |
| HR-G03 / P1 — role/position/access separation | PRODUCT GAP — DO NOT IMPLEMENT؛ C HIGH | sensitive side effects explicit و granular guard QA، no capability reset |
| HR-G04 / P1 — credentials authority | DEFERRED / CONDITIONAL؛ C HIGH broad single reset gate | eligible/privileged target policy و sensitive send/storage separation؛ runtime vulnerability/exploit NOT VERIFIED |
| HR-G05 / P1 — impersonation policy/audit | DEFERRED / CONDITIONAL؛ C HIGH session switch | purpose/action restrictions، durable start/end log، expiry/session recovery؛ not inherently readonly |
| HR-G06 / P1 — temporal hierarchy | PRODUCT GAP — DO NOT IMPLEMENT؛ C/G | apply-now intervals versus business effect، close+insert failure/race QA؛ missing history UNKNOWN |
| HR-G07 / P1 — compensation integrity/authority | DEFERRED / CONDITIONAL؛ A/C | granular visibility/edit و paid-period locks/overlap/rollback؛ no transaction proof across paths |
| HR-G08 / P1 — failed apply recovery | PRODUCT GAP — DO NOT IMPLEMENT؛ C | successful branch≠verified DB effects؛ partial/unknown/idempotency/negative tests |
| HR-G09 / P2 — bulk/export semantics | DEFERRED / CONDITIONAL؛ C | server IDs/fields/scopes، sensitive export parity و truthful outcomes؛ no destructive defaults |
| HR-G10 / P2 — onboarding identity/completion | NEEDS VALIDATION؛ A/C | duplicate/missing linkage، complete access activation و scheduling proof |
| HR-G11 / P2 — Request HR entry / chain bypass | NEEDS VALIDATION؛ C | HR create form absent current context؛ final authority on pending_review policy explicit؛ no invented mandatory complete chain |
| HR-G12 / P3 — documents/training | NEEDS VALIDATION؛ columns C | complete document/training workflow not established؛ no scope expansion by assumption |

OPD-02 escalation/ACK، OPD-06 currency/tolerance، OPD-08 maker/checker و OPD-09 entitlement routing binding/deferred. No new SLA/payroll/HR legal lifecycle assumption. **Design blocker ندارد** برای scoped workforce/request/history و guarded current domains؛ activation حساس و successful role-specific behavior هنوز pending evidence/QA است.

## 19. Compatibility / Safe Adoption

**KEEP + ADAPT**. Preserve user/workforce IDs، current/ historical hierarchy، role/capability mappings، request/approval/compensation/audit history، sales attribution و current workflows/routes تا explicit rollout.

| Adaptation | Compatible bridge / future acceptance |
|---|---|
| Consolidated Workforce editor | same fields/IDs/legacy mappings، sensitive deltas explicit؛ no destructive merge |
| Structure/Access inspector | position/permission/role/credential separated؛ existing mappings محفوظ، no forced role remap |
| Temporal/change preview | old/new/current history و requested/applied times؛ no historical-manager backfill |
| Request/impact/handover | current statuses/chain IDs محفوظ؛ final actual effect/failed state honest؛ OPD-05 policy before activation |
| Restricted credentials/impersonation | presentation separation، current gates documented؛ granular policy future rollout/QA |
| Compensation history | unit/period/snapshot preserved، paid-period/rollback guards validated؛ no earned-credit rewrite |
| Imports/exports/bulk | roundtrip/schema/provided fields maintained؛ source/server scope/field parity و partial results |

No destructive user rewrite، permission reset، forced role remap، historical manager rewrite، audit deletion یا past sales credit rewrite. **FUTURE IMPLEMENTATION — REQUIRES QA / ROLLBACK** برای risky work. rollback نباید existing approvals/credential events/financial attribution/history پاک کند.

Future QA ضروری: real non-Admin HR role read/write/negative guards؛ account/profile identity conflicts؛ cyclic/inactive/scope/effective-date parent changes؛ close+insert and access partial failure؛ concurrent request/change و definitive applied result؛ transfer/termination protected open work و auth timing؛ privileged credential targets/SMS partial/log redaction؛ impersonation scope/action/audit/expiry؛ compensation overlap/payroll-lock/failure rollback؛ sensitive export/filter/server scope و per-row bulk truth. هیچ این write tests یا implementation اکنون اجرا نشده.

## 20. Claude Design Handoff

### WHAT CLAUDE MAY ASSUME

HR Mission و شش domain اصلی IA Freeze‌اند؛ Workforce، Onboarding/Transfer، Requests، Structure/Access، authorized Compensation و Exceptions/Audit. Credentials/Impersonation separate restricted policy، Import/Bulk/Export secondary conditional. Gate0 temporal/identity/permission/history binding؛ code inventory authoritative static، runtime HR unverified. IDs/routes/history/legacy mappings preserved.

### WHAT CLAUDE MUST NOT INVENT

Admin access=HR LIVE VERIFIED؛ HR unrestricted WP Admin؛ position=role=permission=credential؛ disable user=complete handover؛ intermediate approval=applied change؛ scheduled transfers/terminations یا complete history بدون proof؛ past attribution inferred current parent؛ uniform credential privileges یا readonly impersonation؛ salary lock/rollback success از UI؛ new payroll/training/document/SLA workflow؛ all-success bulk یا secret exposure.

### SHARED COMPONENTS TO REUSE

Role identity/scope contracts، hierarchy context، request step/reason/history، action result/partial/system states، sensitive field visibility، governed report/export context و employee/account linkage references. Shared role design foundations visual فقط؛ Sales customer/Case ownership یا Finance actions وارد HR نشده‌اند.

### HR-SPECIFIC DESIGN NEEDS

Workforce Explorer؛ Employment Timeline؛ Hierarchy Change Preview؛ HR Request Review Queue؛ Handover Impact Panel؛ Access State Matrix؛ Sensitive Action Confirmation؛ HR Audit Timeline. WHAT: subject/old-new/effect/outcome/owner/limits، نه layout/component visual. Sensitive confirmation business impact آگاهانه نشان دهد؛ approval policy جدید invent نکند.

### LIKELY NEW SHARED COMPONENTS

Requested-versus-applied change context؛ effective-interval history؛ access source/override inspector؛ protected-work handover impact؛ identity linkage conflict؛ sensitive action outcome/audit context. Reuse خارج HR با policy مستقل؛ هیچ database schema/service/component implementation تجویز یا اجرا نشده.

### DEFERRED / CONDITIONAL ITEMS

Real HR runtime، OPD-05 handover، temporal failure/race/scheduled-effect policy، credential target authority و send-secret boundaries، impersonation purpose/write/audit/expiry، compensation/payroll locks/rollback، failed apply recovery، bulk/export fields/security و documents/training scope. Conditional authority یا successful runtime را با تصمیم بصری نهایی نکنید.

**HR PRODUCT SPEC FROZEN — READY FOR CLAUDE REDESIGN**

**NO CODE CHANGED.**
