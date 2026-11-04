# Slice 15 — Biawin and fulfillment readiness

D-35a–e and D-38 are **unresolved**. No authoritative decision record on `main` changes that. The
decision register in `CRM-STANDALONE-TARGET-ARCHITECTURE-V1.md` §32 lists both as open, and nothing in
`docs/` records an approval.

**No live activation or wallet charging is authorized.**

## 1. Adapter classification (VERIFIED)

| Kind | What exists | Evidence |
|---|---|---|
| Synthetic adapter | `integration/biawin.SyntheticPort`. Deterministic test behaviours: ACK, REJECT and three timeout variants. Test-only. | `internal/integration/biawin/port.go`; e2e tests in `internal/integration/biawin/*_e2e_test.go` |
| Disabled adapter | `integration/biawin.DisabledPort`. `Enabled() == false`; dispatch returns `ErrPortDisabled` and changes nothing. | `port.go`; `providerops.go:307` |
| Actual provider integration | **NONE.** No HTTP client, credentials or endpoint configuration in the target. crm-api reaches no port; no command depends on the provider package. | `TestNoProviderInComposedSystem`, `TestAPIServerHasNoLegacyDependency`; CI Slice 14 boundary step |
| Real provider test evidence | **NONE.** | — |

Legacy client (evidence only, never called): `plugin/current/includes/class-sn-biavin-api.php`.
- `create_order` and `charge_wallet` are HMAC-signed calls.
- Each request gets a random `X-DCW-Request-ID`, with a 25 s timeout.
- There is no retry, no lookup by reference and **no callback contract** (`:43-81`).

## 2. Readiness items

| Item | Status | Evidence / gap |
|---|---|---|
| Service catalog availability | Schema exists; **ships empty** | `internal/catalog`; no admin route or CLI |
| Service definition approval | **None recorded.** Every definition needs an `approval_ref`. | `catalog/registry` |
| Subscription activation contract | SHADOW only. DB CHECKs pin `rollout_mode = SHADOW_COMPARE` and `SHADOW_ACTIVE`. CONTROLLED → BLOCKED (D-35). | Slice 14 matrix §6; `00003_biawin_activation.sql` |
| Activation event producer | **None.** billing emits no `InvoiceFullyPaid`. | Slice 14 matrix §6 |
| Per-service fulfillment ownership | Implemented: catalog SUPERVISOR/EXECUTOR scopes, no HR hierarchy. Scopes are empty. | Slice 14 matrix §5 |
| Real Biawin connector | **NOT AVAILABLE** | §1 |
| Customer PII transfer | Legacy sends name, **mobile** and email (`class-sn-biavin-api.php:49,61`). No privacy-approved minimal payload exists. | Product + Security decision needed |
| External API authentication | Legacy HMAC with an API key stored in a legacy table (`sn_project_api_configs`, no spec). Target secret handling: none. | Secret store required; credentials must never be migrated |
| Provider idempotency | **Unproven.** Legacy uses a random request id per call. Target uses its own stable reference plus write-ahead UNKNOWN. | `providerops.go` |
| Timeout / UNKNOWN reconciliation | Implemented in the library and tested synthetically. **No process runs it** (U6). | `providerops.go:394` |
| Callback protocol and authenticity | Synthetic envelope v1, HMAC-bound, strict canonical decoding (fixed in `26f92b1`). **Not a Biawin protocol.** No callback route. | Slice 14 matrix §9 |
| Operational case assignment and completion | Implemented for target-native SHADOW cases. Legacy cases are read-only. | Slice 14 tests |
| Financial dependencies of services | Components with `CONSUMES_CREDIT` / `CHARGES_LEDGER` → BLOCKED (D-35). Wallet charge → BLOCKED (D-38, DB CHECK). | Slice 14 matrix §12 |

## 3. Classification

| Dimension | Classification |
|---|---|
| Integration readiness | **NO-GO** (no connector, no provider contract) |
| Business decision readiness | **NO-GO** (D-35a–e, D-38) |
| Operational readiness | **NOT VERIFIED** (no relay or reconcile process, empty catalog) |
| Data readiness | **NOT VERIFIED** (subscription parity synthetic only; project subscription tables unspecified) |

## 4. Evidence required before any Biawin activation

1. Owner decisions on D-35a–e and D-38, recorded on `main`.
2. A Biawin provider contract covering:
   - a stable client reference with idempotency semantics
   - lookup by reference
   - a signed callback specification, mapped onto envelope v1 inside an adapter
   - error taxonomy and rate limits
3. A privacy-approved customer payload with no phone as a key.
4. Secrets provisioning in a secret store, plus rotation.
5. A provider sandbox test run with recorded evidence, including timeout-after-create and duplicate-callback
   cases.
6. Relay and reconcile worker processes with backlog alerting.
7. Owner-approved catalog definitions and per-service scopes.
