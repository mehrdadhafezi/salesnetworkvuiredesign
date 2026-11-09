# R4 — Commission and entitlement reconciliation

**No commission is posted, approved or settled**, and no legacy calculation changed.

R4 adds `ReconcileEngines` (`internal/commission/domain/reconcile.go`) on top of the Slice 13 shadow
model:
- the entitlement identity (`EntitlementIdentity.Key`, INV-032);
- the lifecycle;
- the parity checks.

## Verified properties

| Requirement | Evidence |
|---|---|
| Stable entitlement identity | Key = invoice / stage / purpose / share facet / context. The recipient is an attribute, never part of the key (`TestReconcileEngines`; Slice 13 tests) |
| Historical credit ownership | A recipient disagreement is a finding (`RECIPIENT_DISAGREEMENT`), never a correction from the current hierarchy |
| Original sales attribution | Six attribution facets are kept distinct (Slice 13 model) |
| Calculation basis | Slice 13 policy snapshot plus exact decimal arithmetic. R4 adds no basis |
| Explicit amount and unit | `Exact{Hundredths, Unit}`; unit disagreements are classified, never converted |
| Duplicate engine prevention | Classified, not resolved: `MULTI_ENGINE_DUPLICATE` (two legacy engines, one key) and `SAME_ENGINE_DUPLICATE` (non-idempotent retry) |
| No payment / wallet posting in SHADOW | A shadow credit claiming POSTING_REQUESTED / POSTED / SETTLED / REVERSED / CORRECTED → `SHADOW_ECONOMIC_EFFECT`. `Approve` / `RequestPosting` stay decision-blocked; lint `s13-posting-dispatch` |
| Preview, approval, posting, settlement distinct | `ByState` counts each lifecycle state separately |
| E1–E4 without assuming one engine | E1–E4 plus the CRM shadow are all sources; none is preferred; the shadow beside one legacy engine is a comparison, not a duplicate |
| No approved key → no reconciliation key | Non-invoice (OPD-09) and component (D-36) credits are reported in `Unkeyed` with the reason |
| Idempotency | Same input in any order → the same report (`TestReconcileEngines`). The Slice 13 PostgreSQL shadow-run replay test (`TestCommissionShadowRun`: same input → replayed run, policy versions not duplicated) still passes |

## Not decided here

- OPD-09 (engine routing, non-invoice keys, correction);
- D-36 (bundle component split);
- D-41 (posting basis);
- OPD-08 (approval).

No posting engine is activated.
