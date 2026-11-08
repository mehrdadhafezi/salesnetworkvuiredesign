# R2 — Legacy write-back contract (E-3)

**Status: real legacy write-back BLOCKED.** No suitable approved legacy endpoint exists.

This document records three things:

1. the evidence for that conclusion;
2. the adapter boundary R2 implements;
3. the exact legacy-side contract that would have to be built, reviewed and approved before any real write-back can run.

## 1. Evidence: canonical legacy 2.0.160 (`plugin/current`, unchanged)

| Write needed | Legacy path | Authentication | Usable by a service? |
|---|---|---|---|
| Contact outcome | `wp_ajax_sn_update_lead_status` → `ajax_update_lead_status` (`class-sn-plugin.php:305`), `SN_Seller_Flow::apply_activity` | Logged-in WordPress session cookie plus a `sn_public` / `sn_admin` nonce (`check_ajax_referer`, 71 call sites) | **No.** It would require reusing a user session (forbidden). |
| HR request create | `admin_post_sn_hr_change_request_create` (`:103`, `:10429`) | Logged-in session plus form nonce | **No** |
| HR request review | `admin_post_sn_hr_change_request_review` (`:104`, `:10530`) | Logged-in session plus form nonce | **No** |
| Manual payment | Seller payment form handlers | Logged-in session plus nonce. Financial. | **No.** It also needs a financial contract. |

The plugin's only REST route is the read-only `sn-crm/v1/campaign/report` (`class-sn-campaign-tracking.php:1411`), which uses a partner API token. There is no authenticated, idempotent, machine-to-machine write API.

Writing legacy SQL tables directly is forbidden. Modifying `plugin/current` to add an endpoint is out of scope and would be a separately authorised change.

## 2. Adapter boundary implemented in R2

```
integration.writeback_command ──Worker.Step──▶ writeback.Connector (neutral intents)
   └─ legacy/writeback.Connector (maps intents to 2.0.160 semantics; network-free)
        └─ Legacy port (store-level operations)
             ├─ legacytest.SyntheticLegacy  (deterministic, tests only; lint-enforced)
             └─ legacy/compat.Client        (PROPOSED contract client; cannot be built: ErrNotApproved)
```

| Requirement | Where |
|---|---|
| Typed operation contracts | `lwb.LeadOutcome`, `HRCreate`, `HRDecide`. Compat operations: `lead_outcome.apply`, `.observe`, `hr_request.create`, `.create.observe`, `hr_request.decide`, `.decide.observe` (DB CHECK). |
| Explicit source command ids | `request_token` = target command id (lead: dashless; HR: the command UUID) |
| Idempotency keys | The same token. Legacy must treat a repeated token as the same effect (2.0.160 already has a UNIQUE `request_token` for seller-flow events). |
| Expected source state | `expected.status` and `expected.legacy_request_id` on HR decide. Legacy must refuse with `STATE_CONFLICT`, never overwrite. |
| Authentication abstraction | `compat.Credentials`. HMAC-SHA256 implementation with distinct client and server keys (≥ 32 bytes, never printed). |
| Evidence-preserving records | `integration.writeback_exchange` (migration `00012`): append-only; request and response SHA-256, HTTP status, verdict, outcome, correlation, timings. No bodies, keys or PII. Evidence is recorded **before** the answer is used. |
| Strict validation | HTTPS only, no redirects, 16 KiB bound, signed 200 response, `DisallowUnknownFields`, no trailing data, echo of contract / operation / token, enum outcome and reason, reference format |
| Correlation ids | `lwb.WithCorrelation` (set by the connector from the intent) and sent in every request |
| Safe timeout | A timeout or transport error is `ErrOutcomeUnknown` and the command becomes UNKNOWN. The next step is an observation, never a re-apply. |
| DISABLED mode | `CRM_WORKER_WRITEBACK_ADAPTER=disabled` (default). `writeback_deliver` is gated BLOCKED; commands stay PENDING; `writeback_backlog` reports them. |
| Deterministic synthetic adapter | `legacytest.SyntheticLegacy` (fault injection: timeout before or after commit, observe down, duplicates) |
| Separate production configuration | `CRM_WORKER_WRITEBACK_ADAPTER=compat` fails startup. `compat.New` requires `approvedContract`, an empty constant, so activation needs a reviewed code change citing the approval and the endpoint evidence. CI checks the pin stays empty. |

## 3. Write-back state machine (existing vocabulary, unchanged names)

```
PENDING ──claim (write-ahead)──▶ UNKNOWN ──apply answer──▶ APPLIED | REJECTED | UNKNOWN
UNKNOWN ──observe COMMITTED──▶ APPLIED
UNKNOWN ──observe ABSENT──▶ PENDING          (only proven absence permits a retry)
UNKNOWN ──observe AMBIGUOUS──▶ RECONCILIATION_REQUIRED
UNKNOWN ──unreadable × budget (R2)──▶ RECONCILIATION_REQUIRED (WRITEBACK_UNKNOWN, DOMAIN_OWNER_REVIEW)
WAITING_DEPENDENCY ──dependency APPLIED──▶ PENDING
SHADOW_COMPARE: PENDING ──observe──▶ COMPARED (never writes)
```

Invariants (tested):

- one logical operation key;
- no second apply without proven absence;
- no success after a timeout or after an unverifiable 200;
- the original source identity and legacy references are preserved;
- rejection never rewrites target history;
- no financial delivery in the worker allowlist;
- legacy remains master: no dual-master write, no overwrite of a legacy change (`STATE_CONFLICT`).

## 4. Missing legacy-side contract (proposal for separate review)

`sn-crm-writeback-compat/0.1-proposed`. **A proposal, not an existing API.**

- One authenticated endpoint, `POST` over HTTPS. Request body:
  `{contract, operation, request_token, correlation_id, expected{}, payload{}}`.
- Headers: `X-SN-Key-Id`, `X-SN-Signature` (HMAC-SHA256 over the exact body).
- Legacy-side duties:
  - verify the signature;
  - enforce idempotency on `request_token`;
  - check `expected` against current state;
  - run the **same** business rules as the interactive handlers (owner-only lead update, archive refusal, one open HR request, reviewer-only decisions);
  - write its own audit row.
- Response: HTTP 200 only, signed with the server key.
  `{contract, operation, request_token, outcome: APPLIED|REJECTED|OBSERVED, legacy_ref, reason, matches, state{}}`.
- Rejection reasons:
  - `OWNER_MISMATCH`
  - `ARCHIVED`
  - `NOT_FOUND`
  - `DUPLICATE_OPEN_REQUEST`
  - `STATE_CONFLICT`
  - `NOT_AUTHORIZED_REVIEWER`
  - `VALIDATION_FAILED`
- Observe operations are read-only and return the effect for the token.
- Required before approval:
  - Product and Security approval of this contract;
  - the legacy implementation and its review;
  - key management;
  - a staging legacy copy with the endpoint;
  - a pinned endpoint and TLS identity;
  - an UNKNOWN reconciliation rehearsal on staging;
  - a rollback plan.

## 5. Blocked items

| Item | Blocker |
|---|---|
| Real lead / HR write-back | No legacy endpoint; contract not approved (E-3) |
| Financial write-back (manual payment) | Separately approved financial contract required |
| Staging rehearsal of delivery | Needs the compatibility endpoint on a legacy staging copy |
