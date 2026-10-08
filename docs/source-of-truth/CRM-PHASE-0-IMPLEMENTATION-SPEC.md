# CRM PHASE 0 IMPLEMENTATION SPEC V1

تاریخ: 2026-10-07 · **ENGINEERING SPECIFICATION ONLY — IMPLEMENTATION NOT STARTED**

## 1. Scope

Phase 0 زیرساخت ایمنی و تشخیص خواندنی را آماده می‌کند: **A artifact/runtime baseline؛ B boot/read purity؛ C result؛ D native/source identity؛ E financial dependency؛ F permission context؛ G audit/correlation؛ H query metadata؛ I feature-gate interface؛ J tests**. این سند محدوده قابل کدنویسی بعد از approval است؛ task فعلی فقط Markdown Specification است.

Authoritative inputs: **P** [Foundation Plan](CRM-IMPLEMENTATION-FOUNDATION-PLAN.md)، **X** [Cross-role Architecture V1](CRM-CROSS-ROLE-ARCHITECTURE-V1.md)، **G** [Gate 0](GATE-0-PRODUCT-INVARIANTS.md)، **C** [Code Registry](CODE-EVIDENCE-AND-ACTION-REGISTRY.md)، **S** [Status Catalog](STATUS-TRANSITIONS.fa.md)، **A** [Audit پذیرفته‌شده](CRM-PRODUCT-ROLE-ARCHITECTURE-AUDIT.fa.md). Role contracts دوباره طراحی نشده‌اند؛ clarification مالی از قرارداد مصوب G/X/P کافی است. Seller/Supervisor source limitation و مبنای پذیرفته‌شده در X unchanged.

**KEEP + ADAPT؛ NO MIGRATION**. Scope: additive internal value contracts، read-only adapters، diagnostic comparison، isolated tests. Excluded: Seller/new role UI، rollout shell، permission remapping/enforcement switch، canonical ID persistence، F01 write-fix activation، Finance posting/HR apply changes، commission migration، new REST/AJAX endpoint، DB/table/index/cap/page setup، Git/GitHub setup، live audit. Current handlers/boot/guards remain untouched.

CODE VERIFIED برای seamهای exact پایین؛ proposals **INFERENCE / HIGH** based on P/X/G؛ runtime installed bytes/negative/concurrency behavior **NOT VERIFIED**. **HR ROLE RUNTIME NOT LIVE VERIFIED؛ FINANCE ROLE RUNTIME NOT LIVE VERIFIED.** [Compact evidence / integrity checks](../evidence/phase-0-spec-validation-2026-10-07.md). نام‌های پیشنهادی internal هستند، نه کلاس/API/DB موجود یا implementation commitment.

## 2. Existing Code Seams

Treatment: KEEP=current retained؛ CALL=read operation فقط بعد purity/scope qualification؛ WRAP=adapter query/context، نه command execution؛ EXTRACT LATER=ممنوع در Phase0؛ AVOID=no diagnostic call؛ TEST ONLY=isolated spy/static fixture، نه live handler. Private methods با reflection برای production دور زده نمی‌شوند.

| Current file / method | Responsibility / side effects | Phase 0 treatment | Risk / required test |
|---|---|---|---|
| [sales-network.php](../source/sales-network-v4/sales-network.php):sn_bootstrap_load_core :140 | includes migration/HR/hierarchy/scope/reports/invoice/plugin؛ MIS_Report::register registers hooks | KEEP؛ TEST ONLY hook inventory؛ no real-plugin bootstrap in pure runner | required files may register writes/cron؛ BootGraphTest records registrations without firing lifecycle |
| bootstrap::sn_bootstrap_plugin_instance :155 / should_run_full :119 | lazy singleton + optional run؛ CLI/cron/admin/AJAX route contexts | KEEP؛ AVOID as contract-loader | boolean run=false isn't full boot purity guarantee؛ LightFullDispatchCompatibilityTest |
| bootstrap::sn_bootstrap_shortcodes :165 / register_light_shortcodes :204 | shortcode→renderer mapping؛ Dot marketing attrs forwarded | KEEP؛ TEST ONLY route/alias/attr fixtures | handler override/double registration/attribute loss؛ LegacyCompatibilityTest |
| [SN_Plugin](../source/sales-network-v4/includes/class-sn-plugin.php)::__construct :32 / run :39 | constructor registers Dot hooks؛ run registers actions/init/admin_init/setup/callbacks | KEEP؛ AVOID construction for query-contract tests؛ TEST ONLY registration graph | init schema/caps/pages/cleanup/background calls؛ ReadPurityTest |
| SN_Plugin::sn_portal_page_registry :7506 / page_url :7652 | options/permalink route resolution and defaults | KEEP؛ WRAP read context only later； no ensure_required_pages | defaults vs actual URLs، missing page ≠ create now؛ RouteResolutionFixtureTest |
| [Scope Service](../source/sales-network-v4/includes/class-sn-scope-service.php)::get_viewer_profile / get_viewer_position_slug / get_enforcement_mode / should_enforce | HR/current hierarchy、legacy position fallback؛ off/audit_only/enforce_*، finance/Admin exceptions | CALL/WRAP captured read snapshot after tests؛ don't set modes | rank/view exception falsely grants action؛ PermissionContextTest across current modes |
| [HR Service](../source/sales-network-v4/includes/class-sn-hr-service.php)::get_profile_by_user_id :68 vs get_or_create_profile_for_user :50 | getter SELECT + local cache؛ get-or-create inserts profile | getter CALL qualified؛ get-or-create AVOID/TEST ONLY | diagnostic account/profile creation؛ mutation spy rejects write |
| [Hierarchy Service](../source/sales-network-v4/includes/class-sn-hierarchy-service.php)::get_current_parent_profile_id :13 / parent_user_id :21 / descendants :87 | current relationship readers | CALL/WRAP scoped fixture snapshot | missing relation/cycle/depth coverage ≠ zero or historical truth؛ HierarchyContextTest |
| Hierarchy::assign_parent :29 / end_current_assignment :66 | closes old then inserts/logs actual relation؛ success branch not transaction proof | KEEP؛ AVOID؛ EXTRACT LATER for write family | HR partial apply/interval effects； test static seam plus denial of any write call |
| [SN_Invoice](../source/sales-network-v4/includes/class-sn-invoice.php)::get :43 | SELECT by ID؛ constructor reads gateway/credential options | WRAP minimal invoice projection via read port؛ no credentials in context | pure lookup constructor pulls secrets unnecessarily؛ NativeInvoiceReadAdapterTest field whitelist |
| SN_Invoice::update_status :51 / request_payment :58 / gateway verify methods | write/external payment effects | KEEP؛ AVOID/TEST ONLY call-blocking | resolver must never issue/verify/finalize payment |
| SN_Plugin::sn_distribution_returnable_items_for_actor :20731 | private candidate SELECT with history/status filters + tables-ready helper | KEEP؛ TEST ONLY existing predicate against snapshots؛ no command eligibility reuse | candidate list omits universal financial resolution/unknown handling؛ F01ShadowFixtureTest |
| SN_Plugin::handle_distribution_return_items :20758 / handle_mis_return_assignments :22053 | return/reset paths؛ MIS handler calls schema repair/migrate even before run-mode evaluation | KEEP؛ AVOID even dry_run؛ EXTRACT LATER guard integration | dangerous dry-run/replay; NoHandlerExecutionTest |
| [Seller Flow](../source/sales-network-v4/includes/class-sn-seller-flow.php)::get_state :248 / ensure_schema :151 / tables :54 | get_state invokes ensure_schema; missing states triggers installer؛ states/events store last_invoice_id/invoice refs | AVOID getter؛ WRAP pure SELECT adapter over existing schema checks; private tables map evidence | get isn't pure؛ missing table must yield UNKNOWN, no CREATE/ALTER؛ SellerFlowAdapterPurityTest |
| [Dot Flow](../source/sales-network-v4/includes/class-sn-dot-flow.php)::tables :216 / get_case :4012 / payment_rows :5802 | private native cases/links/payments readers | WRAP pure native SELECT adapter؛ don't expose/call private methods | missing links/cases/payments vs no dependency؛ DotAdapterCoverageTest |
| Dot::invoice_conversion_payment_context :5812 / financial_cancellation_guard :6889 | invoice conversion link→Case/payment context؛ guard specific cancellation policy؛ missing link table branch can return allowed | KEEP؛ TEST ONLY known conversion projection؛ not universal no-dependency proof | empty summary/allowed≠full source absence؛ DotUnknownCoverageTest |
| Dot::register_hooks :57 / maybe_upgrade / due workers | init/admin_init upgrade + SMS/archive workers | AVOID in runner؛ TEST ONLY intercepted hook graph | boot can produce business/maintenance effects independent of query |
| MIS native sn_mis_data_rows/pool/distribution relations؛ [Report Source](../source/sales-network-v4/includes/reports/class-sn-mis-report-source.php)::__construct :6 / rows :66 / hydrate :72 | auth/people/current scope؛ gathers rows→pool/items/flow/invoice/histories؛ direct wpdb/log diagnostics | WRAP bounded source-link reads؛ full hydrate TEST ONLY source coverage comparison | report row grain not global Case؛ SQL errors and bounds unknown؛ MISNativeRefTest |
| SN_Plugin::render_wallet_box_for_user :35850 / wallet_autopost helpers :35743… | renderer calls seller/supervisor autopost؛ engine/data guards conditional | KEEP؛ AVOID read supplier؛ TEST ONLY eligible isolated spy | F07 conditional credit attempt؛ ReadPurityTest must detect eligible path |
| [Report Service](../source/sales-network-v4/includes/reports/class-sn-report-service.php)::definitions :14 / filters :25 / query :68 | legacy raw reports with passed seller scope | KEEP؛ WRAP metadata/fixture comparison | legacy-only leads/preinvoice grain؛ F02/03/04 contract tests |
| [Executive Report](../source/sales-network-v4/includes/reports/class-sn-report-executive.php)::definitions :24 / filters :40 / query :200 | shared scoped metrics/query | KEEP؛ TEST ONLY fixture parity / CALL pure read if verified | same label different cohort/scope/predicate؛ no formula change now |
| [MIS Report](../source/sales-network-v4/includes/reports/class-sn-mis-report.php)::register :8 / ajax :67؛ Store::install :10 / step :89 / cleanup :179 | report job/cache/schema/cron state writes | KEEP؛ AVOID install/step/ajax in zero-write tests؛ TEST ONLY instrumentation | technical writes declared separately؛ gather-current not historical-as-of |
| [Helpers](../source/sales-network-v4/includes/class-sn-helpers.php)::send_json :465 | success/message merge، clears output buffers، wp_send_json exit | KEEP؛ AVOID internal contract serializer؛ TEST ONLY old response capture | global response change/output exit؛ LegacyJsonCompatibilityTest |
| bootstrap::short invoice link :275 / init report caps/schema and page setup | missing token save؛ report caps/schema setup، missing password page creation | KEEP؛ AVOID live navigation/bootstrap； TEST ONLY synthetic missing metadata | security/access write isn't harmless cache؛ BootMutationIsolationTest |

No existing method modified/called against live site. Proposed adapter queries derive exact known native columns from C/G and seams above؛ they must not fall back to a state-changing helper when schema/data absent. Full dependency coverage versus conversion-only Dot context remains explicit.

## 3. Proposed Modules

Value objects pure (validation/serialization only)، resolver/ports read-only، no service locator singleton/autoload hooks on inclusion. Combine closely related small values per bounded file below؛ interfaces only where I/O substitution tests require them.

| Proposal | Purpose / inputs → outputs | Must Not Do | Dependencies / consumers / tests |
|---|---|---|---|
| CRM_Context | captured actor/role/position/environment/clock/request correlation→immutable diagnostic context | load WP boot/create profile/read secrets/grant authority | injected snapshots؛ all services؛ ContextIsolationTest |
| CRM_Result / CRM_Item_Result | known outcomes/effects/refs→validated aggregate/per-item model §5 | infer business success from HTTP or mutate domain status | pure refs/reasons؛ all diagnostics؛ ResultContractTest |
| CRM_Native_Ref / CRM_Source_Ref / CRM_Case_Ref | domain IDs/provenance/logical view→valid internal refs §7 | canonical persistence/phone merge/token identity | no I/O؛ adapters/results؛ SourceRefTest |
| CRM_Ownership_Context | proof-backed custody/original/next/event/credit/source/request/reviewer facts→distinct contextual view | derive historical ownership from today's hierarchy | snapshot refs/time/proof؛ permission/report consumers؛ OwnershipContextTest |
| CRM_Permission_Context | captured route/module/action/row/field/actor states→observed current and target diagnostic differences | replace guards/enforce/toggle cap or scope mode | current observation adapter؛ tests/compare only؛ PermissionContextTest |
| CRM_Query_Metadata | query contract + observation/bounds→honest metadata §11 | compute trusted metrics or invent totals/freshness | native/scope refs/clock؛ all query results؛ QueryMetadataTest |
| CRM_Audit_Context | actor/action/object/time/correlation/result→redacted internal trace §14 | write audit table/options/PII or claim durability | injected correlation/sink test port؛ ephemeral diagnostics؛ AuditRedactionTest |
| CRM_Financial_Dependency_Resolver / Result | refs+required adapter coverage+captured reads→deps/blocked/unknown/conflict §8 | return/reset/reassign/relink/create/finalize invoice/status change | read port + four adapters؛ F01 diagnostic tests/shadow |
| CRM_Case_Resolver / Resolution_Result | native refs + evidence→RESOLVED/RELATED/UNKNOWN/CONFLICT §9 | persist aliases/create canonical IDs/phone proof | read-only evidence graph؛ source/dep/report diagnostics؛ CaseResolverTest |
| CRM_Metric_Query_Contract | declared grain/source/cohort/time/scope/predicate ID/unit→metadata validation | implement/refactor KPI formulas | pure contract/no metric evaluation except fixtures؛ F02/03/04/09 tests |
| CRM_Feature_Gate | proposed interface inputs→disabled/enabled reason decision §13 | choose storage/register options/change business flag/grant permission | default-disabled pure policy/provider stub؛ harness only； FeatureGateTest |
| CRM_Read_Port / CRM_Dependency_Adapter | bounded allowed table/column query projection→facts + coverage/error | arbitrary SQL/write/repair/query installer/fallback command | injected SQL/query spy؛ native adapters؛ Purity/Coverage tests |

Names are proposals؛ no new abstraction for unrelated SMS/gateway/UI/HR apply. Primary new production logic in Phase0 is **diagnostic only**؛ successful evaluation is not command authorization. Proposed namespaces/classes must pass symbol conflict check before future coding.

## 4. File Placement

Minimal additive proposed `includes/foundation/`؛ current explicit `require_once` style، PHP≥8.0 baseline. No Composer/autoload migration mandatory. Proposal keeps contracts and bounded native adapters together until evidence warrants separate folders. Paths below **not created**.

| Proposed file inside plugin | Bounded contents |
|---|---|
| includes/foundation/class-crm-result.php | CRM_Result + CRM_Item_Result + status/reason validation |
| includes/foundation/class-crm-refs.php | Native/Source/CaseRef + alias evidence values |
| includes/foundation/class-crm-context.php | Context + Ownership + Audit values/redaction |
| includes/foundation/class-crm-permission-context.php | observed permission context and diagnostic comparison |
| includes/foundation/class-crm-query-metadata.php | Query_Metadata + Metric_Query_Contract validation |
| includes/foundation/interface-crm-read-port.php | fixed read operations and coverage contract؛ no generic command executor |
| includes/foundation/class-crm-native-read-adapters.php | Legacy/V4/SellerFlow/Dot dependency + MIS source projection adapters؛ allowlisted query map |
| includes/foundation/class-crm-financial-dependency-resolver.php | read-only union/status/coverage result |
| includes/foundation/class-crm-case-resolver.php | proof graph diagnostic resolution |
| includes/foundation/interface-crm-feature-gate.php | gate contract + disabled baseline/test provider؛ no persistence |
| tests/foundation/bootstrap.php / run.php | explicitly include pure modules only، deterministic offline runner؛ never require sales-network.php |
| tests/foundation/fixtures/native-facts.php | synthetic value/query fixtures؛ no inserts/PII |
| tests/foundation/*Test.php | named suites §16؛ query/hook/external-effect spies |

No existing 248-file reorganization، no source file copying into parallel writer. Initially foundation files loaded **only by isolated test runner**؛ no additions to plugin bootstrap/shortcodes/action registration. Later diagnostic integration requires its own accepted purity seam؛ cannot count runtime exposure by importing all legacy modules casually. Production diagnostic storage/hook instrumentation not introduced in Phase0. If actual supported runtime differs PHP/WP، baseline gate precedes deployment.

## 5. Result Contract

Internal value serialization (not AJAX migration): `{status, items, reason_codes, correlation, metadata}`؛ status one of **SUCCESS / PARTIAL / FAILED / UNKNOWN / CONFLICT / UNAUTHORIZED / BLOCKED**. `correlation` bounded nonsecret opaque string or null with explicit absence؛ never invented historical ID. `metadata` typed allowlisted diagnostics، includes evaluation type/effects-known/coverage when applicable؛ no arbitrary DB row/secret.

| Status | Precise interpretation |
|---|---|
| SUCCESS | requested diagnostic evaluation complete، or hypothetical command intent proven applied/idempotent unchanged؛ subject/evaluation_type mandatory distinction |
| PARTIAL | multiple item/effect outcomes known mix or evaluation incomplete coverage؛ unknown/unprocessed explicit؛ no all-success claim |
| FAILED | known execution/evaluation failure؛ zero business effect only if proof says so؛ not timeout after possible commit |
| UNKNOWN | outcome/commit effects cannot be determined؛ retry reconciliation required for writes |
| CONFLICT | fresh facts/state/proof contradict intent or each other؛ no invented resolution |
| UNAUTHORIZED | observed denied route/module/action/row/field؛ Phase0 maps fixtures/current guard observations, does not replace endpoint authority |
| BLOCKED | known safety/policy barrier، e.g. financial dependency; no command executed |

Per-item exact shape:

```text
item_ref: NativeRef | CaseRef | null (null requires missing-ref reason)
status: one of the seven internal result statuses
reason_code: stable diagnostic code | null
message: safe descriptive string | null (not machine predicate)
correlation: same operation reference | null
before_ref: native/event/snapshot reference | null
after_ref: native/event/snapshot reference | null
metadata: allowlisted map {evaluation_type, disposition?, coverage?, ...}
```

`before_ref/after_ref` are references، not copied sensitive records or new snapshots persisted. For diagnostic read they normally null؛ evidence refs metadata separate. Optional `disposition` for future bulk accounting APPLIED/SKIPPED/REJECTED/FAILED/UNKNOWN/UNPROCESSED، not extra business status. Phase0 never claims APPLIED financial/HR effects. Disposition SKIPPED may accompany SUCCESS only with known unchanged/no-op reason، not hide missing source.

Aggregate: homogeneous complete evaluations→SUCCESS unless evaluation requested blocked/conflicting/denied outcome itself； dependency resolver evaluated successfully can have `evaluation_result=SUCCESS` while **diagnostic_verdict=BLOCKED**، not «return succeeded». mixed/incomplete→PARTIAL with coverage، uncertainty→UNKNOWN unless mixed known and unknown items then PARTIAL+unknown list؛ conflicts/denials retain per-item exact class. Consumer must inspect verdict/coverage for dependency، not generic success flag. Transport success independent؛ HTTP=200/status string/current helper `success=true` proof business commit نیست. No external JSON/HTTP shape change now.

## 6. Reason Codes

Small stable foundation-only taxonomy؛ domain backend states unchanged. Multiple reasons may coexist; deterministic sorting/dedup، no relying on translated message or label. Unrecognized raw source state preserved with mapping UNKNOWN، not coerced to active=false.

| Reason code | Meaning / expected use |
|---|---|
| DEPENDENCY_FINANCIAL_ACTIVE | valid protected invoice/stage/payment/financial obligation exists؛ diagnostic block |
| DEPENDENCY_PROTECTED_DOWNSTREAM | materialized/fulfillment/source obligation known؛ no ordinary reset |
| DEPENDENCY_UNKNOWN | missing/ambiguous target، adapter/schema/coverage insufficient؛ fail closed |
| LINKAGE_CONFLICT | contradictory proven relation/source domain evidence |
| STALE_STATE | captured version/time/context no longer matches required fresh observation |
| ROW_SCOPE_DENIED / ACTION_DENIED / FIELD_DENIED | specific observed policy layer rejection؛ no grant from reason name |
| RECIPIENT_INELIGIBLE | known inactive/wrong relationship/action context؛ assignment not performed |
| OUTCOME_UNKNOWN | possible effect unresolved؛ no blind write retry |
| SOURCE_UNAVAILABLE | known query/schema/read failure؛ empty isn't success |
| COVERAGE_INCOMPLETE | bounded/unprocessed/missing-domain evidence؛ no negative proof |
| POLICY_UNRESOLVED | OPD-bound decision cannot be inferred |
| FEATURE_DISABLED | feature-gate default/conditional exposure off |
| CONTRACT_INVALID | internal value shape/invariant validation fails before I/O |

Reason taxonomy version initially proposal `foundation-v1`، not plugin schema version. Codes diagnostic، never save into invoice/HR/Case status columns. Future additions backward compatible with consumers accepting code+unknown mapping؛ persisted reason migration excluded.

## 7. Source / Case Refs

Internal DTO/testing shapes only؛ IDs serialized decimal strings to avoid lossy cross-language numeric assumptions. Existing native DB IDs retain original meaning; positive decimal validation for those domains، no virtual-ID sign/encoding reinterpretation unless proven adapter. opaque native refs elsewhere require declared domain policy. Null/UNKNOWN/N/A distinct through explicit quality fields؛ no zero sentinel minted as real entity.

```text
NativeRef = { domain, id }
  domain: LEGACY_LEAD | V4_DISTRIBUTION | MIS_ROW | MIS_POOL | DOT_CASE |
          INVOICE | PAYMENT_STAGE | PAYMENT | LEDGER_TX | EVENT | ...declared extension
  id: native identifier string (no token/phone); invalid domain/id -> CONTRACT_INVALID

SourceRef = { native_ref, origin_batch_ref?, origin_actor_ref?, origin_at?, provenance_state }
  provenance_state: VERIFIED | INCOMPLETE | UNKNOWN | NOT_APPLICABLE
  origin fields: existing references only; missing reason tracked; not today's owner/time

CaseRef = { native_refs, source_refs, canonical_id, resolution, evidence_refs }
  canonical_id: null in Phase0; existing proven canonical external ref only if actually present,
                never created, generated or persisted by this module
  resolution: RESOLVED | RELATED | UNKNOWN | CONFLICT
  native_refs: nonempty refs; logical grouping proof doesn't require stored canonical ID

AliasEvidence = { from_ref, to_ref, relation, proof_level, proof_refs, adapter, observed_at }
  relation: SAME_CASE | RELATED_DISTINCT | FINANCIAL_LINK | SOURCE_MATERIALIZATION | UNKNOWN
  proof_level: AUTHORITATIVE | VALIDATED_LEGACY_FALLBACK | DISCOVERY_ONLY | UNKNOWN
```

Actor/employee/request/batch refs use declared domain extensions as needed، not counterfeit Cases. Financial link doesn't automatically mean two invoice/Case subjects SAME_CASE؛ relationship type proof applies. IDs equal in different domains unequal؛ phone/name/time can appear only redacted discovery context outside NativeRef ID. MIS row≠batch/file، SourceRef≠current custody. CaseRef with no canonical_id may represent **resolved logical view of proven native aliases**؛ no consumer may treat null as unrestricted scope or «no Case». Serialize deterministic domain-qualified strings/keys for tests، not DB schema or identity encoding commitment.

## 8. Financial Dependency Resolver

Proposed input:

```text
{ subject: CaseRef or nonempty native/source refs,
  operation_kind: diagnostic ordinary-return/reset/reassignment/cleanup context,
  context: CRM_Context, required_domains: explicit adapter coverage set,
  policy_refs: accepted G/X versions, observation_token?: existing snapshot reference }
```

Adapter interface proposal `inspect(subject, read_port, context) -> AdapterObservation`؛ no command method. Observation: `{adapter_id, links, missing_targets, conflicts, coverage, evaluated_at}`. Read port exposes allowlisted bounded lookups of native source/links/invoices/stages/payments، not arbitrary SQL or write service. Context actor scope validated before projection; comparison on fixture facts never queries unauthorized live rows.

Dependency exact logical object:

```text
{ subject_ref, target_ref, relationship, evidence_refs, adapter_id,
  source_state: {namespace, raw_value, meaning_quality},
  protection: PROTECTED | NO_ACTIVE_DEPENDENCY | UNKNOWN | CONFLICT,
  amount_context?: {total, paid, remaining, unit, basis, quality},
  obligation_kind: FINANCIAL | MATERIALIZATION | FULFILLMENT | UNKNOWN,
  evaluated_at }
```

Amount values native precision-preserving strings/null، no new conversion/rounding; OPD-06. `NO_ACTIVE_DEPENDENCY` per resolved target isn't whole Case eligible؛ cancelled release OPD-03 remains protected/conditional. Invoice fully paid still ordinary reset protected. Historical event link must resolve current target state; latest link alone insufficient.

Output `CRM_Financial_Dependency_Result`:

```text
{ dependencies[], reasons[], diagnostic_verdict: BLOCKED | UNKNOWN | CONFLICT | NO_DEPENDENCY_OBSERVED,
  coverage: {required[], inspected[], not_applicable[], unavailable[], bounded, complete},
  confidence: VERIFIED_WITHIN_DECLARED_COVERAGE | INCOMPLETE | UNKNOWN | CONFLICT,
  evaluated_at, contract_version, correlation }
```

`NO_DEPENDENCY_OBSERVED` allowed only complete proof within declared applicable domains، never `RETURN_ALLOWED`؛ ordinary scope/reset/recipient/handoff rules still outside resolver. Known protected→BLOCKED even if other sources unknown (unknown retained in reasons/coverage)، contradictory proof→CONFLICT، no known protected + any missing/ambiguous coverage→UNKNOWN. Missing table/query failure/missing target is unavailable/unknown—not empty absence. Not-applicable requires proven source topology، not adapter not implemented.

| Required Phase0 adapter | Read evidence / negative coverage |
|---|---|
| Legacy Invoice | real legacy lead→sn_invoices.lead_id + existent native record/invoices؛ virtual compatibility only validated proof؛ no match alone not global negative |
| V4 follow_invoice | distribution item follow_invoice_id resolve invoice even lead_id=0؛ source/pool/MIS lineage read; missing target→UNKNOWN |
| Seller Flow | existing states last_invoice_id + all relevant events invoice_id in correct source_kind/source_id؛ pure SELECT, no get_state/ensure_schema؛ event reference current target checked |
| Dot | existing sn_dot_cases / sn_dot_invoice_links / sn_dot_payments refs؛ assessment_source vs conversion_payment relations preserved؛ stage/payment/raw states not collapsed; all relevant records, not LIMIT1 summary negative |

Required four adapters implemented/tested diagnostically in future Phase0، no handler activation. TODO coverage: invalid/unproven virtual-ID encodings، implicit/phone-only links، non-invoice bonus/entitlement sources OPD-09، unproven fulfillment/cross-system obligations/Complaint-Biawin native relationship evidence. Mark TODO unavailable or unknown whenever relevant؛ never global «no dependency» while required evidence unavailable. Generic all-CRM completeness not achievable merely because four adapter tests pass؛ output declares exact source coverage. Adapter paging/bounds preserve missing tail; no source repair to fill gaps.

READ ONLY hard prohibition: reset/return/reassign/relink/create invoice/change status/finalize/approve/refund/post/SMS/schema/cap/page mutation. Future F01 commit guard integration **EXTRACT LATER**; Phase0 demonstrates diagnostic block known fixture only، not fixed runtime F01 or authorization.

## 9. Case Resolver

Input native/source refs + explicit AliasEvidence observations؛ output `{resolution, proven_groups, related_refs, unresolved_refs, conflicts, evidence_refs, coverage, evaluated_at, correlation}`. No durable key or alias write. Algorithm contract: validate domains/refs→inspect required native lineage→accept same-case edges only if authoritative/validated compatibility→group proven aliases→retain related-distinct and unknown→report conflicts/coverage. Not metric formula or customer merge.

| Proof level | Condition |
|---|---|
| RESOLVED | relationship SAME_CASE proven by validated source materialization/explicit lineage identity contract؛ all claimed grouping edges proof-backed |
| RELATED | explicit relation links objects but doesn't prove same business Case؛ retain distinct refs, no alias collapse |
| UNKNOWN | missing target/schema/domain proof/ambiguous continuation؛ keep original refs and required-domain gap |
| CONFLICT | mutually inconsistent authoritative relations/domain mapping؛ do not choose newer/current hierarchy implicitly |

Phone/name/time proximity **never RESOLVED**؛ same invoice linking two refs alone isn't blanket proof those Cases identical. Source materialization relation must establish continuity according G/OPD-01; related invoice vs native business identity distinguished. Partial proven groups coexist unknown refs; overall coverage incomplete، not all-resolved claim. CaseResolver tests wrong-domain same IDs، missing link، contradictory proof، phone-only hint، related-distinct relation؛ no records merged.

## 10. Permission Context

Exact captured structure:

```text
{ actor_ref, acting_role_names[], position_ref/slug, route_ref, module,
  action, row_refs[], field_names[],
  ownership_context, current_reviewer_ref?, current_state_refs[],
  current_guard_observations: [{layer, guard_ref, observed_decision, evidence_ref}],
  target_diagnostic_decisions: [{layer, concept, ALLOW|DENY|UNKNOWN, reasons}],
  scope_version?, evaluated_at, correlation }
```

Roles/position/cap/credential distinct؛ no credentials serialized. Route/Module/Action/Row-Field layer observations do not produce WP cap changes. Read original policy precedence and legacy fallbacks، retain unknown scopes؛ actor eligible view doesn't make all lower-level actions allowed. **Phase0 observe/map/compare only**؛ any ALLOW is diagnostic relative to captured rule، not trusted enforcement token or grant. Guards still executed by legacy handlers as before; no new commands/routes wired. Diagnostic needed data from injected minimal snapshots، not unauthorized data probes.

F06 detection fixture: finance-view allowed + action=rule/create/run review/flag/post؛ current handler observation accepts view، target action authority unproven→`VIEW_WRITE_COUPLING` diagnostic metadata + ACTION_DENIED/UNKNOWN decision as policy states، not runtime exploit claim. F08 fixture: simple position/profile update changes access/compensation/credentials؛ display action/field effect classification and target separate-authority gap، no automatic revoke/role reset. Rank-only Deputy extra-number review or Senior ready assignment denies under X； current reviewer and manager_id row ownership independent of ancestor visibility. Inactive/missing profile + legacy role fallback coverage explicit.

Future command wrapper can enforce only after explicit acceptance/negative tests and near-commit fresh checks؛ Phase0 context cache cannot authorize later write. Documentation/fixtures label current observations vs target requirements distinctly.

## 11. Query Metadata

Exact proposed metadata:

```text
{ source: [{domain, adapter, native_scope_ref?, source_version?, availability}],
  grain, scope: {actor_ref, scope_kind, definition_ref, policy_version?, completeness},
  cohort: {definition_ref, filters_ref, coverage},
  time_basis: {mode: CURRENT_SNAPSHOT|EVENT_RANGE|HISTORICAL_AS_OF|EVENT_COHORT_CURRENT_GATHER,
               event_field?, range?, cutoff?, timezone, gather_window?},
  unit: {native_unit?, currency?, basis?, quality} | NOT_APPLICABLE,
  freshness: {observed_at?, source_updated_at?, quality},
  coverage: {domains[], complete, unavailable[], unprocessed_count?},
  version: {contract, adapter, source?}, evaluated_at,
  bounded: boolean, loaded_count: nonnegative integer,
  total_count: nonnegative integer | null, total_quality: KNOWN|UNKNOWN|NOT_APPLICABLE }
```

Known total_count may exceed loaded_count؛ bounded=true with null total valid؛ source unavailable cannot total=0. Query version/source age unknown doesn't inherit evaluated_at as last DB update. Scope identifier/hashes no sensitive list public; raw filters exclude PII in logs. HISTORICAL_AS_OF requires history proof، cannot inferred from datepicker or range+current gather. Technical stepped gather may have window rather than instant snapshot.

Metric_Query_Contract requires name/grain/source/cohort/time/scope/formula-or-predicate-ref/unit/freshness/coverage/version؛ nonratio denominator N/A؛ incomplete contract declared NOT FINAL. **No production metric formulas implemented** in Phase0. Fixtures only demonstrate F02 current pre_invoice predicate coverage، F03 legacy-only vs V4 source set، F04 incompatible inputs/results can't be silently equated، F09 as-of versus current gather. Comparison harness compares captured result sets/metadata; doesn't «fix» counts or guess cache/build cause.

## 12. Read Purity

Call-graph verification plan: registered boot hooks→entrypoint→constructors→renderer/query/getter→helpers→DB/meta/options/cap/page/HTTP/SMS/ledger effects. Static graph seeded from exact seams، test spy observes representative paths with missing and eligible data. Capture registration separately from callback execution؛ suppressed callback isn't proof real full path pure.

| Effect class | Definition / Phase0 rule |
|---|---|
| ALLOWED TECHNICAL | declared diagnostic in-memory cache/ephemeral trace؛ technical job/cache writes only isolated future mode with explicit approval؛ strict new contract runner has zero persistent writes |
| BUSINESS MUTATION | financial tx/status/payment/custody/personnel/access/session/token/credential/notification effect؛ prohibited from new read path |
| MAINTENANCE | schema/index/page/cap seed، repair/backfill/cleanup؛ prohibited new read path، existing setup not executed |
| UNKNOWN | uninspected call/callback/provider or possibly side-effectful fallback؛ cannot certify read purity until inspected/isolated |

| Path | Known effect / detection contract |
|---|---|
| bootstrap/init/admin_init | caps/schema/pages/migration/cleanup registrations؛ intercepted registry and deny DDL/options/cap/page writes if callback tested |
| SN_Plugin constructor→Dot register_hooks | register init due-SMS/archive/upgrade؛ object instantiation isolation، no actual WordPress lifecycle fired |
| shortcode→renderer | resolve mapping/attrs with fixtures؛ no actual role panel load in pure contracts runner |
| wallet renderer | F07 autopost for eligible seller/supervisor، matrix guards conditional؛ isolated spy must capture credit attempt when enabled by fixture |
| report job/ajax/store step | metadata/rows/cleanup/cron write؛ classified technical vs maintenance، not zero-write read query |
| HR get_or_create | profile INSERT; rejected before DB call؛ pure getter fixture still works |
| Seller get_state | missing table→ensure/install schema؛ new adapter reports source unavailable and no DDL |
| MIS return dry_run/read-like paths | repair/migrate and reports/options possible؛ handlers forbidden; pure native adapter separate |
| short invoice link | missing access token UPDATE؛ forbidden production diagnostic call، token/login identifiers never serialized |

Test infrastructure uses deny-write read port + intercepted WP mutation/provider functions، captures query types and hook attempts. Must catch raw `$wpdb` bypass too via recording connection proxy/stub in isolated environment؛ read-port guard alone insufficient for old helper. `SELECT` allowlist not arbitrary text prefix allowing multi-statement/write routine； prepared fixed operation keys and read-only DB credentials in future qualified snapshot integration if supplied. No real DB inserts/DDL/schema initialization in Phase0 tests.

**Acceptance: new foundation read tests cause zero BUSINESS MUTATION and zero unapproved DB/cap/page/DDL writes؛ technical writes declared and isolated، strict mode zero persistence.** Negative control legacy eligible render must attempt prohibited credit, tester intercepts before effect؛ positive pure adapter reads same seeded facts without attempt. Mutation spy failure is evidence detection works، not request to fix legacy renderer in this phase. Unexpected boot effect→test fails/no-go new path، existing plugin unchanged. Business no-write≠technical metadata no-write؛ both separately reported.

## 13. Feature Gates

Proposed interface `decide(FeatureGateInput) -> FeatureGateDecision`:

```text
Input: {feature, role, module, path_type: read|write, actor_ref, environment, cohort_ref?}
Output: {enabled: boolean, disabled_reason: reason_code|null}
```

Validation invalid/missing context disabled CONTRACT_INVALID؛ unknown feature/cohort or unauthorised target context disabled؛ disabled baseline FEATURE_DISABLED. Phase0 defaults all role/UI/command exposure disabled؛ tests may inject true decision for **pure diagnostic feature fixture** only، no actual flag stored or route activated. Read/write decision independent، multiple roles actor resolved explicitly not highest-rank. Enabled flag never substitutes authorization/state/dependency guard; permission comparison separate.

No storage selection (DB option/file/environment) or current option change؛ no new flags created. Existing scope/commission/purpose/posting options not repurposed. Inclusion/constructor/interface has no WP hook registration، get_option/update_option or seed effect. Test gate decision deterministic with same inputs؛ disabled write cannot fall back to uncertified legacy command merely because old UI exists.

## 14. Audit / Correlation

Context exact shape `{correlation_id, actor_ref, action, object_refs[], reason_codes[], result_ref_or_summary, timestamp, time_basis?, evidence_refs[]}`؛ actor-role snapshot where available؛ reason/result no secret/raw evidence. Inject deterministic correlation/clock into tests؛ no historical token minted and no storage side effects. Before/after references extensions، correlation separates operation linkage from business idempotency key.

| Current store / evidence | Reference capability | Phase0 treatment |
|---|---|---|
| activity/financial history/payment review events | actor/object/time/reason current existing refs | read correlation to native events؛ do not append in Phase0 |
| distribution item logs / Seller flow events | from/to/action/source/invoice/time refs | diagnostic reference linkage from native event IDs |
| HR request approval_path / assignment history | reviewer/subject/change/actor/context refs | captured current trace؛ absent correlation UNKNOWN؛ no chain rewrite |
| commission run/item logs + wallet tx metadata | run/item/tx/approval/note refs | keep stable key links； never audit helper that also credits |
| MIS import/run/job/source/history reports | batch/run/row/source/result refs | read-only projection، persisted job not invoked |
| New diagnostic operation | no durable existing record guaranteed | **ephemeral in-memory Phase0 correlation**؛ not durable audit/idempotency guarantee |

Stores **can supply references** now; accepting new correlation values later requires field/caller/authority validation, not assumed schema support. No unified audit table/column or log writes needed. Comparison output uses safe IDs/counts/fingerprints، not actual customer/bank/HR data or full SQL filters. Areas missing durable correlation remain explicit in result metadata؛ Phase0 completion doesn't certify write-retry tracking.

## 15. Fixtures

Synthetic minimal arrays/object projections + query/hook/effect spies؛ no production PII، tokens/gateway credentials/SMS recipients، no real users created or data inserted. Fixed clock/timezone، deterministic native ID strings and table-prefix placeholders. Fixture graph separate native/source/financial/status/permission/time facts； values serve tests، not invented business policy.

| Fixture ID | Given | Expected diagnostic acceptance |
|---|---|---|
| FX-L0 | Legacy ref، no invoices، all applicable sources inspected | NO_DEPENDENCY_OBSERVED scoped only؛ not command eligible |
| FX-L1 | Legacy real invoice lead relation active/preinvoice | protected BLOCKED |
| FX-V4 | V4 follow_invoice_id valid invoice with lead_id=0 | F01 known danger BLOCKED، source refs preserved |
| FX-MULTI | follow invoice + flow event/history other valid active invoice | union both؛ last link alone not enough |
| FX-MISSING | invoice target absent or required table missing | UNKNOWN/DEPENDENCY_UNKNOWN؛ no schema install |
| FX-CONFLICT | incompatible authoritative source/case link proof | CONFLICT، no phone/newest-based winner |
| FX-DOT | assessment_source materialized Case or conversion_payment+stage/payment refs | protected dependency + explicit Dot relation domain |
| FX-DOT-GAP | Dot summary empty but missing payments/links table | incomplete UNKNOWN، cancellation allowed branch not reused as absence proof |
| FX-HR-INACTIVE | inactive/missing subject/profile with legacy position fallback | eligibility/context explicit؛ no profile creation/no rank grant |
| FX-SCOPE | identical role with owned vs out-of-scope row / allowed vs restricted field | captured row/field diagnostic difference، no unauthorized serialization |
| FX-F06 | view-only Finance accepted current write guard | view/write coupling detected، target write authority not assumed |
| FX-F08 | profile/position edit implies access/compensation/credential effects | separate sensitive delta/context gap، no cap remap |
| FX-F02 | invoice raw state pre_invoice; old count excludes it | metadata/predicate difference observable؛ no prod formula correction |
| FX-F03 | legacy IDs none + V4 ref present | Legacy Leads Only 0 not Total Cases0 |
| FX-F04 | two same-name metrics different scope/cohort/source inputs/counts | NONCOMPARABLE context in comparison metadata؛ root cause not guessed |
| FX-F09 | event cohort/time range + gather-current outcome | not HISTORICAL_AS_OF؛ gather window/coverage |
| FX-F07 | eligible wallet legacy engine, and matrix-enabled/ineligible variants | eligible branch intercepted credit attempt؛ variants can't falsely certify all reads |
| FX-REFS | same integer across two domains، phone-only hint، related-distinct refs | identities distinct؛ no RESOLVED by discovery |
| FX-BOUNDS | adapter limited/paged/tail unknown، loaded<known total | bounded incomplete/no false negative dependency proof |
| FX-RESULT | mixed known/unknown outcomes, unchanged-idempotent vs new effect | exact item/aggregate distinctions؛ no fake all-success |
| FX-BOOT | missing schema/page/cap/token + callback registrations | mutation spy observes hypothetical attempt before side effect؛ no actual creation |

Fixture statuses use S raw values with namespace، OPD-dependent outcome policy unknown؛ no actual refund/return/posting/apply. Snapshot parity from synthetic/captured sanitized definitions، not copied private production DB.

## 16. Tests

Named future test suites / acceptance، **not executed or created in this task**. Runner PHP8-compatible/isolated explicit includes؛ external PHPUnit choice optional later, no dependency install or WordPress auto-installer now. Test categories include unit، isolated integration، negative permission، read-effect and snapshot parity.

| Test suite | Category / pass criteria |
|---|---|
| FoundationResultContractTest | unit؛ all 7 statuses、items/ref/null validation、Partial+unknown، transport distinction، dependency verdict≠operation success |
| SourceRefTest | unit؛ deterministic domain-qualified native serialization، origin missing explicit، wrong-domain same IDs distinct، no token/phone IDs |
| OwnershipContextTest | unit؛ current/original/next/event/credit independent؛ hierarchy update doesn't rewrite past |
| FinancialDependencyResolverTest | unit + read-adapter integration؛ FX-L/V4/MULTI/DOT/GAP/BOUNDS union/protection/coverage/errors؛ missing→unknown fail-closed diagnostic |
| CaseResolverTest | unit graph؛ RESOLVED vs RELATED vs UNKNOWN vs CONFLICT، proof-edge validation؛ no persistent aliases |
| PermissionContextTest | unit/negative fixtures؛ four layers/current vs target، F06/F08/rank/current reviewer、no actual guard replacement/grant |
| MetricQueryContractTest / QueryMetadataTest | unit/parity؛ required grain/time/scope/unit metadata، totals null vs zero/bounds، F02/03/04/F09 classification؛ formulas only fixtures |
| ReadPurityTest / BootGraphTest | isolated effect integration؛ detects eligible F07/token/get-or-create/ensure schema/setup attempts and rejects them； new adapter zero write/hook/provider attempts |
| LegacyCompatibilityTest | isolated contract؛ baseline shortcodes/attrs/deep-link/tab map + JSON success/message/redirect semantics unchanged؛ no new prod endpoint |
| AuditRedactionTest | unit؛ no credential/token/PII/raw bank proof، missing durable correlation explicit، no persistence |
| FeatureGateTest | unit؛ unknown/default disabled؛ read/write split؛ flag does not grant action authority/store settings |
| ShadowParityHarnessTest | captured fixtures؛ both ID sets + metadata compared؛ semantic source/bounds differences classified، no legacy write execution |
| ArtifactBaselineContractTest | manifest/static؛ plugin header/runtime-schema/build separately، accepted hashes + actual runtime unknown honest؛ no boot to gather baseline |

Read adapter integration uses fixture query port and optional **already-provisioned read-only sanitized snapshot** later if supplied؛ no automatic DB create/seed/migrate. Negative legacy mutation tests intercepted stubs، not real transaction rollback sandbox؛ rollback doesn't make performed mutation read-only. Raw `$wpdb`/HTTP/WP functions monitored if legacy seam exercised؛ private paths can be inspected/static fixture-mapped until safe isolation is proven. All required adapter behaviors tested even if no full WP harness used؛ lack actual DB snapshot coverage marked validation limitation، not guarantee installed runtime.

## 17. PR Breakdown

Future small PR boundaries only؛ no branch/PR/git setup now. Baseline/tests dependency earlier than contracts shipping؛ read-purity tests scaffold PR-00 and extended later، not deferred until unsafe adapters written.

| Proposed PR | Files / purpose | No-go scope | Tests | Rollback / dependencies |
|---|---|---|---|---|
| PR-00 baseline + minimal harness | tests/foundation/bootstrap.php、run.php、fixtures/native-facts.php、ArtifactBaselineContractTest.php؛ static manifest/runtime unknown checklist | loading real plugin، live DB/runtime audit، migrations/fixtures inserts/Composer/Git setup | manifest/header/value + boot isolation self-test | remove only additive harness؛ deployed artifact evidence supplied before Phase0 deployment； no runtime wiring |
| PR-01 result + refs | foundation/class-crm-result.php、class-crm-refs.php؛ FoundationResultContractTest/SourceRefTest؛ extend minimal fixtures | bootstrap edits/HTTP endpoints/persistent canonical keys/business commands | value/invalid/unknown/serialization tests | delete additive contracts/tests only؛ PR-00 |
| PR-02 read port + dependency adapters | interface-crm-read-port.php、class-crm-native-read-adapters.php、class-crm-financial-dependency-resolver.php； Dependency/Coverage/Purity tests | return/reset/reassign/relink/status/schema、legacy handler even dry-run、full Dot/flow boot | four adapters fixtures/error/bounds/F01 blocked + deny-write spy | remove unconnected files； PR-01 + read-purity scaffolding |
| PR-03 case diagnostic | class-crm-case-resolver.php + CaseResolverTest؛ proof graph/related grouping | alias persistence/canonical ID/phone merge/financial relink | proof/conflict/missing-domain tests | unconnected module removal؛ PR-01/02 |
| PR-04 permission/query/context/gate contracts | context/permission/query-metadata/interface-feature-gate files + tests؛ captured diagnostic facts, ephemeral audit | replacing guards/toggling cap/scope/flag/registering new endpoint/metric prod formulas | F06/F08/scope/metadata/ownership/audit/gate defaults | package-only files rollback؛ PR-01/03; P/X concepts |
| PR-05 purity seam instrumentation | tests/foundation effect/hook/query spies، expanded ReadPurityTest/LegacyCompatibilityTest | production renderer change、boot hook removal/enforcement/ledger remediation | missing schema/token/cap/page/eligible F07 negative + pure adapter positive | tests-only removal؛ PR-00/02/04؛ no real legacy effects |
| PR-06 shadow parity harness | tests/foundation shadow harness + sanitized result captures/contracts docs accompanying tests | dual writer، live query automation/PII export/current metric replacement、UI exposure | ID+metadata+coverage parity/F02/03/04/F09/F01 differential | test artifacts only removal؛ PR-02…05 |

Each future PR requires changed-file allowlist、source invariant refs، evidence/limits، no-bootstrap/no-schema/cap delta check and reviewer acceptance. Default runtime flags/interfaces disconnected؛ package rollback foundation only. Production observability/durable correlation or wiring read routes not smuggled into PR-05/06؛ separate authorization/acceptance after Phase0.

## 18. First Coding Task

**FIRST FUTURE CODING TASK: isolated harness + pure result/reference contracts، no plugin integration.** PR-00 ثم PR-01 reviewable separately؛ a single first task may prepare both under bounded file list if user approves، but merges remain small. This confirms actual include-based architecture allows test-only explicit modules without bootstrap edit؛ F01 write activation first PR ممنوع.

Exact initial allowed file proposal:

- `includes/foundation/class-crm-result.php`
- `includes/foundation/class-crm-refs.php`
- `tests/foundation/bootstrap.php`
- `tests/foundation/run.php`
- `tests/foundation/fixtures/native-facts.php`
- `tests/foundation/FoundationResultContractTest.php`
- `tests/foundation/SourceRefTest.php`
- `tests/foundation/ArtifactBaselineContractTest.php`

Task acceptance: pure PHP value validation/serialization، synthetic refs/results، deterministic runner، no real-plugin require، no WP bootstrap/action hook/DB/network/file-secret reads، no persistent IDs؛ existing 248 file hashes unchanged aside from separately authorized additive files، zero production registrations. Invalid refs and mixed/unknown outcomes tested؛ native V4/Legacy IDs not rewritten. Rollback delete only these additive unconnected files، no data recovery or schema rollback. ArtifactBaseline records accepted local manifest + actual deployed runtime **UNKNOWN until supplied read-only baseline evidence**؛ first pure contracts can be developed without false runtime claim، deployment waits.

No files above created in this task. Coding begins only after human approval in a later task؛ this deliverable isn't that approval and doesn't initialize GitHub.

## 19. Definition of Done

Phase0 complete **only when all evidence below present**؛ current document marks requirements، not completion of coding/tests:

| Requirement | Acceptance proof |
|---|---|
| Deployed artifact/runtime baseline identified | authorized read-only manifest of actual build/hash/versions/PHP-WP-DB/schema capability and routes/caps references؛ no secrets؛ accepted ZIP label alone insufficient |
| New path no unapproved mutation | inclusion/constructor/query graph test and effect-spy proof zero business/DDL/cap/page/provider persistence؛ technical writes declared/isolated |
| SourceRef/CaseRef tested | domain/provenance/unknown/no phone identity/no canonical persistence tests pass |
| Four dependency adapters tested | Legacy/V4/Flow/Dot bounded native reads/target resolution/coverage/errors؛ TODO domains explicit |
| F01 known dangerous fixture blocked diagnostically | V4 follow invoice + lead_id0→BLOCKED with proof؛ handlers never called؛ not claim live fix |
| Permission context maps current guards | route/module/action/row/field fixtures، F06/F08 gap metadata، no guards changed |
| Metric/query metadata tested | source/grain/cohort/time/unit/bounds/freshness + F02/03/04/F09 distinctions؛ no production formula edits |
| F07-class side effect detection | eligible negative fixture triggers spy blocked credit attempt، new foundation read positive zero attempt |
| Audit/gate contexts safe | ephemeral correlation redacted، defaults disabled/no storage no authority grant |
| Legacy availability/integrity | old routes/shortcodes/action/JSON/native IDs/history/status/permissions untouched، symbol/include collision check |
| No write activation | no new endpoint/hooks/routes/flags/guard enforcement/posting/HR apply/return integration |
| Rollback package-only | additive unconnected code removable without DB/options/caps/data adjustment |

Real HR/Finance accounts not necessary for pure diagnostic contract tests؛ actual role runtime remains NOT LIVE VERIFIED and gates for later activation preserved. Existing unsafe legacy behaviors may remain—they must be **detected/avoided**, not changed under Phase0. Full source coverage not invented to satisfy DoD. TODO relevant domain yields UNKNOWN; limited coverage reported. User approval to code is separate from evidence that completed Phase0 passes DoD.

## 20. Risks

| Concrete Phase0 risk | Detection | Containment |
|---|---|---|
| Include/autoload/symbol duplication | symbol search + require_once repeat tests | bounded proposals/prefix؛ no changing current loader, test-only include |
| Constructor/hook side effects | register-vs-execute call graph、missing schema/token/eligible data spy | inject plain context/read ports، no SN_Plugin/Dot boot in pure runner |
| Read-like get_state installs schema | missing flow table fixture + DDL interception | direct pure adapter/check → UNKNOWN؛ avoid ensure helper |
| Private helpers tempt reflection production calls | code-review allowlist/call graph | native adapter queries from verified columns؛ private methods TEST ONLY/static، no visibility refactor |
| Legacy globals/$wpdb bypass | connection/function spies detect raw writes/options/provider attempts | no real WP/DB bootstrap، fixed query operation keys، readonly snapshot later |
| Hook double registration | deterministic hook registry repeated include/instance tests | foundation files registration-free؛ runtime integration excluded |
| False dependency confidence | missing target/schema/domain/limit/conflict multi-link fixtures | exact declared coverage، unknown fail closed، no allowed-return field |
| Current hierarchy→historical attribution | changed-parent/native-owner fixtures | provenance-specific ownership/time values؛ no backfill |
| Sensitive credential/PII in DTO/fixtures/log | allowlist/redaction tests/static fixture scan | synthetic values، minimal invoice projection، don't serialize raw rows/options |
| Shadow calls legacy writer/dry_run | command/HTTP/write spies، source call review | fixture evaluation only؛ return/report-step handlers never replayed |
| Query performance/coverage limits | bounded read-count/fixture tail tests، actual scale remains unknown | bounded adapters/pages and honest incomplete؛ no unrestricted full hydration |
| Snapshot drift/current vs source facts | evaluated_at/version/domain set comparison | no historical-as-of or live parity claims from synthetic tests |
| Test bootstrap seeds schema/users/caps | effect guard catches installer/run/init callbacks | no real plugin require or WordPress automated install، no DB test seeding |
| Null canonical_id interpreted as no identity | CaseRef consumer contract tests | resolved native grouping independent of persistence؛ null never unrestricted scope |

Risks actual runtime occurrence unverified؛ detection plan is not remediation implementation. No speculative new security-policy checklist added beyond P/G sensitive boundaries.

## 21. Deferred Decisions

| Existing dependency / engineering choice | Phase0 baseline | Deferred boundary / owner |
|---|---|---|
| Artifact/runtime/build provenance | static local hashes + explicit unknown installed build | environment owner supplies authorized read-only baseline before deploy/DoD |
| OPD-01 lineage/identity format | proof-backed native diagnostic graph، unresolved preserved | persistent canonical/alias store not selected؛ Product/MIS/Sales/Engineering |
| OPD-03 release/reset / OPD-04 refund | all current financial dependencies protected/unknown diagnostic | no release/refund authority/proof policy invented؛ Product/Finance/Sales/MIS |
| OPD-06 money boundary | native amount/unit projection، unknown metadata | no conversion/tolerance normalization؛ Finance/Product |
| OPD-08/09 maker/checker/engine/noninvoice keys | context/gaps only؛ no engine/ledger call | sensitive Finance activation later؛ Finance/HR/Product |
| OPD-05 personnel handover / OPD-10 delegation | captured permission/ownership context | no HR apply/role grant/skip-level default activation |
| Full Dot/fulfillment/cross-system coverage | verified native relations four adapters + TODO/domain unavailable | confirm missing topology before global no-dependency claim؛ domain owners |
| Feature gate storage/new API transport | pure disabled interface/internal DTO | no option/schema/REST creation؛ Engineering + release owner later |
| Durable audit/intent/idempotency persistence | ephemeral diagnostic correlation and existing native event refs | existing store write adaptation/additive persistence unapproved، not needed for pure Phase0 |
| Test framework/runtime | simple isolated PHP8 runner proposal، no package installed | use existing approved tooling if discovered later؛ no test boot mutation |
| Full runtime read instrumentation | isolated fixture/static spies | wiring production instrumentation/diagnostic route requires separate approval/purity gate |

Architecture/Product contracts not reopened. Missing policy narrows diagnostic coverage rather than creating new decision IDs duplicating X register. New modules pure/adapters possible without any migration. Phase0 never resolves F01/F06/F07/F08 runtime defects by claiming specification success.

## 22. Freeze Review

| Freeze check | Verdict |
|---|---|
| Bounded additive read-only Phase0 spec possible | YES؛ internal contracts/native diagnostic adapters/test-only inclusion |
| Exact seams/effects identified | YES؛ methods/risks/treatments/tests؛ unsafe getters/handlers explicitly avoided |
| First coding task small/non-UI/non-migrating | YES؛ result/ref values + isolated synthetic harness only |
| Production guards/actions/flags/DB unchanged | REQUIRED and preserved in current specification task |
| Known F01 diagnostic fixture + F07 detection specified | YES؛ runtime fix/activation not included |
| Stored canonical IDs/history/status/schema rewritten | NO |
| Actual Phase0 code/tests/PRs created/executed | NO؛ files/test names/PRs proposals only |
| Coding authorization | NOT GIVEN by this deliverable؛ later explicit user approval required |

Document validation: 22 ordered sections، local source/input links and preserved prior input/code hashes in E. This is specification QA، not execution of proposed test suites or actual Phase0 DoD. **STOP after Markdown؛ do not begin coding.**

CRM PHASE 0 IMPLEMENTATION SPEC V1 READY
SAFE TO BEGIN FOUNDATION CODING AFTER USER APPROVAL
NO CODE CHANGED
NO DB CHANGE
NO GITHUB ACTION
