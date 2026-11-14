# R7-5 — Deterministic integrated synthetic CRM dataset

Status: TEST / DEVELOPER CAPABILITY ONLY. No production code, migration,
schema, flag default, grant, business rule or import block is changed.
Production remains **NO-GO**.

Baseline: `main` @ `7a793f4115742d1a7f6de35f4712709e4d711dce`.
Scope: R7-A audit §5 item 4 and §7 row R7-5. The audit asks for a command
that composes existing synthetic fixtures (`legacytest`) into one coherent
dataset for local runs and E2E, without real data.

## What it is

The dataset is **`legacytest.IntegratedScenario()`**, version `r7-5.v1`, snapshot id
`syn-r7-5-integrated-v1`.
- It is one synthetic legacy snapshot package, composed from the **existing** read-world fixtures. No record is new.
- It is seeded into a **disposable** PostgreSQL database through the **existing** import pathway only.

| Layer | Source fixture (unchanged) | Seeded through (existing code) | Lands in |
|---|---|---|---|
| Identity + organisation | `HRScenario` (users 1–13, hierarchy with findings) | staging loader → `shadow.IdentityCommand` / organisation import | `identity.*` (SHADOW grants only), `organization.*` |
| Sales | `SalesScenario` (leads 101–102, items 501–504) | sales import | `cases.*`, `custody.*` |
| Customers | `CustomerScenario` (1001–1005) + issued-to `2001` (as in the billing suites) | customers import | `customers.*` |
| Billing / payments | `BillingScenario` (invoices 2001–2004, payment 9 → 2999) | billing then payments import | `billing.*`, `payments.*` (legacy evidence only) |
| Biawin / fulfillment | `BiawinScenario` (memberships 501–505, items 601–614) | biawin + fulfillment shadow import | `biawin.*`, `fulfillment.fulfillment_case` etc. (shadow) |
| Finance | `WalletScenario` (wallets 1–4) | finance shadow rebuild | `finance.*` (shadow ledger) |
| Commission | `CommissionScenario` (rules, dry run, tx 100–113) | commission shadow run | `commission.*` |
| Reporting / MIS | — (projection of the owners above) | reporting projections rebuild | `reporting.*` |

### Relationship map

```
wp_user 1..13 ──► identity.user_account ◄── organization.employment ◄── org_assignment (hierarchy + findings)
      │                    ▲    ▲    ▲
      │                    │    │    └── commission.calculation.recipient_user_id
      │                    │    └─────── finance.ledger_account.holder_user_id
      │                    └──────────── custody.assignment.derived_user_id / custody_event.to_user_id
lead / distribution item ──► cases.work_case ◄── custody.assignment ◄── reporting.case_fact / custody_fact
issued-to 1001..1005, 2001 ──► customers.customer ◄── finance.ledger_account.customer_id
invoice 2001..2004 ──► billing.invoice ◄── payments.payment_attempt / receipt ◄── reporting.payment_fact
        │                     ▲    ▲
        │                     │    └── fulfillment.fulfillment_case.invoice_id, commission.historical_entitlement
        │                     └─────── reporting.invoice_fact
        └── memberships 501..505 ──► biawin.subscription / item_invoice_link
wallet 1..4 + wallet tx (wallet + commission fixtures) ──► finance.* shadow ledger, commission.* parity
```

**Documented unresolved references.** These come from the fixtures, are intentional, and are listed in `IntegratedKnownUnresolved`:
- **Invoice `2999`:** BillingScenario payment 9 and BiawinScenario membership 504 name an invoice that is not staged. It becomes UNKNOWN and is never resolved.
- **Invoice `0`:** CommissionScenario dry-run item 2 uses the legacy "no invoice" sentinel for a daily attendance bonus. It is NOT_COMPARABLE.
- **Account `71`:** a phone-derived customer account that is not an imported user. It stays RELATED or UNKNOWN and is never merged by phone.

## Files

| File | Kind |
|---|---|
| `internal/integration/legacy/legacytest/integrated.go` | `IntegratedScenario`, version constants, `mergeEntities` (merges entities shared by fixtures; a duplicate key or completeness mismatch panics) |
| `internal/integration/legacy/shadow/r7_5_integrated_dataset_test.go` | `setupIntegrated` / `seedIntegrated` / `importModules`: one disposable database, the existing per-module environments reused |
| `internal/integration/legacy/shadow/r7_5_dataset_checks_test.go` | determinism, integrity, repeat run, safety, refusals, non-synthetic block, opt-in export |
| `docs/remediation/r7-5-synthetic-dataset.md` | this document |

`legacytest` stays test-only. The architecture lint rule `pgtest-import`
still allows it in `_test.go` files only, so no production binary can
import the dataset.

## How to use it

- **In a Go test:** `e := setupIntegrated(t); b, steps := e.seedIntegrated(t)`. It runs on a `pgtest` disposable database that is dropped when the test ends.
- **For local runs with the existing tool:** export the package, which is opt-in and never overwrites. Then use `crm-legacy-import`. Its loader stages only `data_origin: synthetic` packages and needs dedicated owner-role DSNs.

  ```
  CRM_R75_EXPORT_DIR=/tmp/r7-5 go test -run TestR75ExportIntegratedPackage ./internal/integration/legacy/shadow/
  crm-legacy-import validate /tmp/r7-5
  crm-legacy-import load /tmp/r7-5        # then import-identity, import-organization, import-sales, …
  ```

## Determinism

**Deterministic source:**
- `IntegratedScenario()` is `reflect.DeepEqual` across calls.
- Two package writes are byte-identical: 38 files, content identity `b13a95e7f9d6fae378e0b2d657e37a94765313d11345cb2e3a1fb1daa2e29182`. This value is the same in every local run and in `crm-legacy-import validate`.

**Deterministic database (`TestR75IntegratedDeterministicAcrossDatabases`):**
- Two fresh databases are seeded independently.
- For each of the 129 populated tables, every row is fingerprinted as JSON. The fingerprint excludes runtime columns.
- A generated UUID that is the primary key of a row with a legacy `source_key` is replaced by `table:spec:source_key`. The same is done for `case_id`, labelled through `cases.case_source`. So references are compared by **meaning**. Every other UUID is masked.
- Result: all 129 tables are logically identical.

**Runtime-generated (excluded by name, `r75RuntimeColumns`):**
- **Generated UUIDs:** every primary key and every `run_id` / `batch_id` comes from the modules at runtime. None is a source key.
- **Hashes over UUID-bearing inputs:** `input_sha256`, `facts_sha256`, `snapshot_key`, `finding_key`, `evidence_key`.
- **Ordinals numbered in UUID order:** `finance.finding.finding_no`, `finance.reconciliation.recon_no`, `finance.ledger_entry.entry_seq`. The set of findings and entries is identical; only their numbering follows UUID order.

## Repeat execution (`TestR75IntegratedRepeatRun`)

On the **same** database, the same package is staged again and every import step is re-run:
- **Re-staging** returns SUCCESS and creates no second batch.
- **Payments, biawin, fulfillment, finance and commission** replay.
- **Identity, organisation, sales and customers** re-record the same evidence, and their authoritative rows are unchanged.
- **`*.import_run`** gains at most one run-record row per module.
- **Reporting** adds one projection run. That is the design: a rebuild is a new projection version, and reads use the latest one. Its content is proven identical to the first run.
- **Every other table**, 112 of 129, is logically unchanged.

## Test-database safety

| Control | Evidence |
|---|---|
| Disposable target only | Seeding exists only in `_test.go` on `pgtest.New`, which creates a fresh database plus restricted roles and drops both. There is no seed path to an arbitrary DSN. |
| Synthetic only | `TestR75IntegratedNonSyntheticIsBlocked`: the same rows under `data_origin: unclassified` are **BLOCKED** by the loader, and no batch is staged. The existing `TestNonSyntheticPackageIsBlocked` also applies. |
| Never overwrites | The export refuses an existing path, verified by re-running it. Nothing ever deletes data to reach idempotency. |
| Nothing else written | `TestR75IntegratedSeed` pins the exact set of populated tables. Staging is checked by rule (one table per non-empty entity). Any credential, session, HR request, payment submission, native work case, provider operation, outbox or write-back row, or enabled flag fails the test. |
| No ACTIVE grants | All imported grants are `SHADOW` (also a schema CHECK). There are 0 credentials and 0 sessions. |
| No flag enabled | 0 rows in `platform_featureflags.feature_flag` have `enabled`. |
| Synthetic contact data | `TestR75IntegratedSourceIntegrity`: every phone is a reserved synthetic number (`09000000xxx`, `+98 900 000 0xxx`) or a `TEST-PHONE-…` marker. Every e-mail ends in `@example.invalid`. No credential-like column carries a value. |
| No provider / legacy / financial execution | Only the import and shadow commands run. No provider port, write-back connector or native financial command is constructed. |
| Referential integrity | Source: every invoice and wallet reference resolves, apart from the documented set. Database: 17 cross-module references by generated id resolve (152 values, 0 orphans). |
| Read gates intact | `TestR75IntegratedReadRefusals` covers three cases: reporting with its flag off returns `FEATURE_DISABLED`; a subject without the capability gets 403; an anonymous request gets 403. Reads change no row in billing, payments, finance, commission, customers, cases, custody or identity. |

These reads use the existing development auth stub, so they are **backend-only coverage** and not evidence of production authorization.

### Negative controls (each made locally, never committed)

| Mutation | Result |
|---|---|
| Seed enables the reporting flag | caught (Seed) |
| Seed sets a grant ACTIVE | **impossible**: the `role_grant_shadow_only` CHECK rejects it; Seed also asserts all grants are SHADOW |
| Remove a documented unresolved reference (`0`) | caught (SourceIntegrity) |
| Compose a fixture twice (duplicate keys) | caught (construction panics) |
| Drop the runtime-column exclusion | caught (DeterministicAcrossDatabases) |
| Repeat run rewrites an authoritative row | caught (RepeatRun) |
| Repeat run duplicates a customer | **impossible**: the `customer_source_unique` constraint rejects it |

## Scenarios that cannot be seeded safely (not in this dataset)

| Scenario | Why not | Where it is covered today |
|---|---|---|
| Controlled-write world: Slice 11 HR requests, Slice 12 payment submissions and evidence, contact outcomes | `WritesScenario` / `PaymentsScenario` reuse legacy user ids 1–10 with other meanings and need clean hierarchies and payable invoices. Merging means re-keying records, which would invent relationships. Those records are also created by **native commands**, not by import. | `setupWrites` / `setupPayments` (R7-1, R7-2, R7-4) |
| Native fulfillment work cases (`fulfillment.work_case`) | Created only from a `SubscriptionItemActivated` event. Financial activation never creates one (D-35), and tests inject the event at the inbox boundary. Seeding them would bypass that block. | R7-3 / R7-4 (`operationsScenario`, `injectItem`) |
| Provider operation states | Exist only through the test-only synthetic provider port, from a work case. There is no real provider contract. | R7-3 / R7-4 provider-boundary tests |
| Credentials and sessions | Runtime records created per test by the existing auth helpers; never part of deterministic source data, and never seeded with a password. | R7-1 harness |
| ACTIVE capability grants | Prohibited (D-34 / D-42 / R5-SEC-01 open). | — |

## Remaining governance blockers (unchanged)

- **D-34, D-42, R5-SEC-01:** no production role bundle carries the shadow or write capabilities, and person-level separation of duties is not provable.
- **OPD-05, SD-04:** HR apply is blocked. **D-35, D-38:** financial activation and completion are blocked.
- **Biawin:** the real API contract is absent.
- **Live verification:** HR and Finance runtime are NOT LIVE VERIFIED.
- **Real data:** real legacy packages remain BLOCKED by the loader.

## Readiness for R7-6 and R7-7

- **R7-7** (E2E: reporting / MIS and Biawin read, OP-21 / OP-26): **ready**. This dataset populates every projection R7-7 reads, including UNKNOWN, CONFLICT and INCOMPLETE values that must not render as zero. The browser harness can call `setupIntegrated` + `seedIntegrated` in place of `setupBilling`.
- **R7-6** (E2E: HR request create and step decision; HR read, OP-04 / OP-05): **partially served**.
  - **HR read:** the organisation evidence here includes cycles, self-parent lines and multiple parents.
  - **HR request flows** must keep using `setupWrites`, which has a clean hierarchy, plus the DEF-HR-01 fix. Final apply stays refused.
