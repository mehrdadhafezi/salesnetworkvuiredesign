# Slice 15 — Legacy/target data reconciliation register

## 0. What was and was not run

| Item | Status |
|---|---|
| Production or staging snapshot available to this audit | **NO**. No snapshot is committed (CI "no snapshot content tracked"), and this environment has no production or staging access. |
| Real-data reconciliation run | **NONE.** No cohort has ever been compared on real data (U4). |
| Synthetic reconciliation | **RUN** as part of the baseline suite: `go test ./...` passed 70 packages at `b3bf31b` against local PostgreSQL. The tests stage deterministic synthetic legacy fixtures (`internal/integration/legacy/legacytest`) and assert counts and diffs. |
| Read-only local database check | **RUN.** `scripts/cutover/readiness-evidence.sh db` on a disposable, freshly migrated local database: 0 staged snapshots, 37 staging tables, 0 flags enabled. |

**No discrepancy count below is zero because a comparison passed.** Where no real comparison ran,
every count is `NOT RUN`. Synthetic results prove that the comparison machinery works on fixtures. They
do not describe production data.

## 1. Source coverage (VERIFIED)

The legacy 2.0.160 code references **89** `sn_*` tables (static scan of `plugin/current/includes`).
Extract specs exist for **34** `sn_*` tables plus `wp_user` and `wp_usermeta`, giving 37 specs
(`internal/integration/legacy/spec`, golden files in `spec/testdata`).

**55 legacy tables have no extract spec.** They cannot be reconciled until they are specified,
extracted and staged:

| Domain | Tables without a spec | Impact |
|---|---|---|
| After-sales / complaints | `sn_after_sales_cases`, `sn_after_sales_case_logs` | Complaint branch (journey 12) has no target data at all. |
| Dot conversion | `sn_dot_cases`, `sn_dot_case_logs`, `sn_dot_invoice_links`, `sn_dot_payments`, `sn_dot_marketing_events`, `sn_dot_marketing_leads`, `sn_dot_operational_archives` | Case coverage (INV-020) incomplete; Dot invoices unlinked. |
| MIS / distribution | `sn_mis_data_rows`, `sn_mis_import_batches`, `sn_mis_lead_pool`, `sn_mis_distribution_plans`, `sn_mis_distribution_plan_items`, `sn_mis_distribution_rules`, `sn_mis_distribution_logs`, `sn_distribution_chains`, `sn_distribution_chain_steps`, `sn_extra_number_requests` | Source provenance (INV-003) and MIS totals incomplete. |
| Campaigns | `sn_campaigns`, `sn_campaign_events`, `sn_campaign_sessions`, `sn_campaign_attributions`, `sn_campaign_conversions`, `sn_campaign_partner_campaigns`, `sn_campaign_partner_tokens` | Attribution history incomplete. |
| Seller flow / statuses | `sn_seller_flow_states`, `sn_seller_flow_events`, `sn_lead_statuses`, `sn_activity_logs` | Outcome history and custom status dictionary (INV-017) incomplete. |
| HR | `sn_hr_change_requests`, `sn_hr_compensation_profiles`, `sn_hr_compensation_history`, `sn_hr_profile_logs` | In-flight HR requests and effective-dated history (INV-028) incomplete. |
| Billing / payments | `sn_invoice_wheel`, `sn_payment_reward_claims`, `sn_report_invoice_state_events` | D-40 reward history and invoice state events incomplete. |
| Projects / Biawin / operations | `sn_project_subscriptions`, `sn_project_subscription_contents`, `sn_project_api_configs`, `sn_project_card_upgrade_rules`, `sn_project_commission_rules`, `sn_project_commissions`, `sn_project_manager_products`, `sn_project_manager_sales_routes`, `sn_project_operations`, `sn_project_operations_manager_products`, `sn_project_operations_sales_routes`, `sn_project_operations_supervisor_sales_routes`, `sn_project_messages`, `sn_project_message_reads`, `sn_project_notes`, `sn_operations_form_definitions`, `sn_operations_wallet_codes`, `sn_product_flow_events` | Catalog, routing, engine E4 commissions, wallet codes and operations queue incomplete. |
| WordPress / WooCommerce | `wp_posts` / Woo orders, `wp_options` (feature options such as `sn_purpose_commission_engine_enabled`) | Engine routing history and order coupling unknown. |

Some of these tables may be runtime-created or optional; for example, `SN_Subscription_Plus` is absent in
2.0.160. Determining which exist in production is an operator task (§4).

## 2. Cohort register

Columns follow the request. "Old" means legacy and "New" means target. Every real-data count is
**NOT RUN**.

| Cohort | Source | Snapshot / extract id | Cutoff | Old | New | Matched | Missing | Extra | Conflicting | Unknown | Amount diff (by unit) | Root cause / evidence | Confidence / coverage |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| Customers and source identifiers | invoice issued-to + leads | none | — | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN | n/a | Resolver tested on synthetic fixtures (`internal/integration/legacy/shadow/customers_e2e_test.go`): phone never RESOLVED alone | No real coverage; Dot/campaign customers unspecified |
| Customer aliases / ambiguous | same | none | — | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN | n/a | RELATED/CONFLICT queue only (D-30, OPD-01) | none |
| Cases and ownership | leads, distribution items | none | — | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN | n/a | custody parity tests (`sales_e2e_test.go`) | Dot cases missing (§1) |
| Assignments and transfers | distribution logs, lead history | none | — | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN | n/a | append-only custody events | MIS distribution unspecified |
| Employees and org history | `sn_hr_*`, wp users | none | — | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN | n/a | capability diff and hierarchy findings (`shadow_test.go`) | change requests and compensation unspecified |
| Invoices and payment stages | `sn_invoices`, stages, items | none | — | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN (IRT/IRR kept separate, OPD-06) | `reconcile-billing` on synthetic data (`billing_e2e_test.go`) | Woo orders unspecified |
| Payment attempts and evidence | `sn_payments`, receipt files | none | — | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN | receipt **files** never extracted | links only |
| Wallet balances and ledger | `sn_wallets`, transactions | none | — | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN | derived-vs-legacy diff with reasons (`finance_e2e_test.go`) | D-41 basis open |
| Commissions and recipients | dry runs, items, matrix, rules, wallet tx | none | — | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN | shadow parity classes (`commission_e2e_test.go`) | Engine routing for past windows UNKNOWN; E4 unspecified |
| Subscriptions and items | memberships, items | none | — | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN | n/a | active-subscription parity (`biawin_e2e_test.go`) | project subscription tables unspecified |
| Fulfillment cases | execution cases, shipping | none | — | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN | n/a | Slice 10 import tests | `project_operations` unspecified |
| MIS and reporting totals | projections | none | — | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN | INV-021 parity tests (`reporting_e2e_test.go`) | MIS rows unspecified; OPD-07 |
| External provider references | legacy actions (`remote_status`), wallet codes | none | — | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN | n/a | Slice 14 providerops (synthetic) | no provider lookup contract |

## 3. Conditions before any real reconciliation

1. **Staging and anonymisation policy approved.** E0 ADR #18 (DEFERRED) and D-23 must be decided:
   "Must exist before any real-data import".
2. **Authorised snapshot.** A snapshot is extracted by an authorised operator using the published specs
   (`crm-legacy-import specs`). It carries a manifest, checksums and an explicit cutoff, and is stored
   outside the repository.
3. **Isolated database.** It is loaded into an isolated non-production database. `crm-legacy-import load`
   currently refuses non-synthetic packages (`cmd/crm-legacy-import/main.go:10`). Enabling real loads is
   an implementation change that needs approval.
4. **Reconcile in order.** Each cohort is reconciled in sequence: staging vs manifest, then module import,
   then per-cohort reconcile reports. Results are recorded in this register with snapshot id and cutoff.
5. **No source changes.** No historical source record is changed to force parity. Phone matches are never
   merged.

## 4. Operator verification (not run)

Run read-only against a legacy replica:
- `SELECT table_name, table_rows FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name LIKE '%sn\_%'`
  — confirms which of the 89 tables exist and their approximate sizes.
- Record wp option values for engine routing: `sn_purpose_commission_engine_enabled`,
  `sn_wallet_auto_credit`, `sn_enable_commission_wallet_posting`.
