# RM-E04C — Seller `/me` API Contract Proposal

**PROPOSAL — NOT IMPLEMENTED, NOT APPROVED.** This fragment is deliberately kept **outside** `api/openapi/` so that it cannot reach generated code, the operation registry or runtime (AGENTS.md: keep unapproved alternatives out of runtime). Baseline `main` @ `c0d7f4b`. Production **NO-GO**.

Architecture: [`rm-e04c-seller-authorization-adr-PROPOSED.md`](rm-e04c-seller-authorization-adr-PROPOSED.md) (recommended Option C). Screen designs: [`../security/rm-e04c-seller-screen-designs.md`](../security/rm-e04c-seller-screen-designs.md).

## 1. Rules every operation below follows

| Rule | Source |
|---|---|
| `GET` only. No Seller write appears here | RM-E03 §4; ADR §4.2 |
| Guard order: feature flag → capability (ACTIVE grant) → `CheckRead(ctx, cap, ScopeOwn)` → `Subject.UserID != ""` else 403 `ROW_SCOPE_DENIED` → predicate inside SQL | SR-01…SR-04; C6, C10 |
| Out-of-scope `{id}` ≡ missing `{id}`: same 404 problem body; sub-resources re-apply the parent predicate | SR-06; C8 |
| Unknown query parameters → 400 `CONTRACT_INVALID`; there is **no** phone parameter until GAP-04 is decided | commission precedent |
| Each list has its own cursor sort name (`me.cases.source_key`, `me.invoices.source_key`) | ADR §8 |
| Counts come from the same scoped query; absent → `count.state = ABSENT` with a reason, never `0` | SR-07; AGENTS.md Sensitive Outcomes |
| Other staff identities are never returned; only `is_caller` booleans | ADR §6–§7 |
| Responses carry `Cache-Control: no-store`, `Vary: Cookie` (already global, C7) | SR-12 |
| Every operation must get an `OWN_READ` entry in `operation_scope_registry.yaml` with its basis, and NT-07 / NT-22 coverage, in the same PR that adds it | C9 |

**Why new schemas instead of reusing `CaseSummary` / `InvoiceSummary`:** `InvoiceSummary.seller` and `.issued_by` are `UserRef` objects carrying another user's `legacy_key` and `user_id` (S-STAFF), and `CaseSummary.source` carries batch-level provenance not needed by the Seller. The Seller DTOs are field-minimised projections; every field maps to an existing column (no invented business state).

## 2. Operations (Option C; capability names are candidates only)

| operationId | Path | Capability (candidate) | Basis (M2 rec. 2 selects) | Screen |
|---|---|---|---|---|
| `listMyCases` | `/api/v1/me/cases` | `cases.own.read` | `CUSTODY_CURRENT_HOLDER` | S1 |
| `getMyCase` | `/api/v1/me/cases/{case_id}` | `cases.own.read` | same | S1 |
| `listMyCaseHistory` | `/api/v1/me/cases/{case_id}/history` | `cases.own.read` | same | S1 |
| `getMyCaseContactOutcomes` | `/api/v1/me/cases/{case_id}/contact-outcomes` | `cases.own.read` | same | S1 |
| `listMyCaseAttribution` | `/api/v1/me/cases/{case_id}/attribution` | `cases.own.read` | same | S2 (sub-resource only) |
| `listMyInvoices` | `/api/v1/me/invoices` | `billing.own.read` | `INVOICE_SELLER_AND_ISSUER` (initial) | S4 |
| `getMyInvoice` | `/api/v1/me/invoices/{invoice_id}` | `billing.own.read` | same | S4 |
| `listMyInvoiceStages` | `/api/v1/me/invoices/{invoice_id}/stages` | `billing.own.read` | same | S4 |
| `getMyInvoiceTotals` | `/api/v1/me/invoices/{invoice_id}/totals` | `billing.own.read` | same | S4 |
| `listMyInvoicePayments` | `/api/v1/me/invoices/{invoice_id}/payments` | `payments.own.read` (+ `billing.own.read` gate on the parent) | inherits invoice | S4 |
| `listMyCommissionEarnings` | `/api/v1/commission/me/earnings` | `commission.own.read` | `COMMISSION_HOLDER` | S7 — **exists**, unchanged |

Not proposed (no server source / contract): S2 list, S3, S5, S6. Not proposed until GAP-04: `/me/cases/{case_id}/customer-contact`.

## 3. OpenAPI 3.0 fragment

Paths are written as they would appear in `api/openapi/openapi.yaml`; `$ref`s point at existing components. Only the S1 list and S4 list are written in full; the other operations follow the same pattern and are listed in §2.

```yaml
paths:
  /api/v1/me/cases:
    get:
      tags: [sales]
      operationId: listMyCases
      summary: "The caller's own cases (own scope only; PROPOSAL)"
      description: |
        Requires cases.own.read (candidate). Own scope only: the predicate selected by M2 proposed record 2
        (recommended CUSTODY_CURRENT_HOLDER) is applied inside SQL to the list, the count and any search.
        Rows whose holder is NULL, unresolved, MULTIPLE or CONFLICT are excluded and not counted.
        Never broadened by hierarchy, position or another role. No phone parameter.
      parameters:
        - $ref: './components/parameters.yaml#/Cursor'
        - $ref: './components/parameters.yaml#/Limit'
        - name: code
          in: query
          required: false
          description: Case / lead code, exact or prefix of at least 4 characters, applied inside the own predicate.
          schema: {type: string, minLength: 4, maxLength: 64}
      responses:
        '200':
          description: SUCCESS / PARTIAL (truncated or incomplete facts) / BLOCKED with FEATURE_DISABLED.
          headers:
            X-Correlation-ID:
              $ref: './components/headers.yaml#/CorrelationID'
          content:
            application/json:
              schema: {$ref: '#/components/schemas/SellerCaseListEnvelope'}
        '400':
          $ref: './components/responses.yaml#/Problem'
        '403':
          $ref: './components/responses.yaml#/Denied'
        default:
          $ref: './components/responses.yaml#/Problem'
  /api/v1/me/invoices:
    get:
      tags: [billing]
      operationId: listMyInvoices
      summary: "The caller's own invoices (own scope only; PROPOSAL)"
      description: |
        Requires billing.own.read (candidate). Predicate selected by M2 proposed record 2 (recommended initial
        INVOICE_SELLER_AND_ISSUER). An invoice with seller or issuer NULL is never in scope and never counted.
        Money is per unit and never summed across units; a "financially confirmed" amount is not returned
        (no Finance rule exists).
      parameters:
        - $ref: './components/parameters.yaml#/Cursor'
        - $ref: './components/parameters.yaml#/Limit'
        - name: code
          in: query
          required: false
          schema: {type: string, minLength: 4, maxLength: 64}
      responses:
        '200':
          description: SUCCESS / PARTIAL / BLOCKED with FEATURE_DISABLED.
          headers:
            X-Correlation-ID:
              $ref: './components/headers.yaml#/CorrelationID'
          content:
            application/json:
              schema: {$ref: '#/components/schemas/SellerInvoiceListEnvelope'}
        '400':
          $ref: './components/responses.yaml#/Problem'
        '403':
          $ref: './components/responses.yaml#/Denied'
        default:
          $ref: './components/responses.yaml#/Problem'
components:
  schemas:
    ScopeEvidence:
      type: object
      description: Which ownership fact admitted the rows and the import batch it reflects (SR-11).
      required: [scope_basis, evaluated_as_of]
      additionalProperties: false
      properties:
        scope_basis:
          type: string
          enum: [CUSTODY_CURRENT_HOLDER, INVOICE_SELLER_AND_ISSUER, INVOICE_SELLER, INVOICE_ISSUER, INVOICE_VIA_CASE_HOLDER, COMMISSION_HOLDER]
          description: Only the basis approved under M2 proposed record 2 may appear at runtime.
        evaluated_as_of: {$ref: './components/schemas.yaml#/Timestamp'}
    ScopedCount:
      type: object
      required: [state]
      additionalProperties: false
      properties:
        state: {type: string, enum: [EXACT, ABSENT]}
        value: {type: integer, format: int64, minimum: 0, description: Present only when state is EXACT.}
        reason: {$ref: './components/schemas.yaml#/ReasonCode'}
    CallerFacet:
      type: object
      description: Ownership facet as seen by the caller. Never carries another user's identity.
      required: [facet, resolution, is_caller]
      additionalProperties: false
      properties:
        facet: {type: string, maxLength: 40}
        resolution: {$ref: './components/customers.yaml#/ResolutionState'}
        is_caller:
          type: boolean
          nullable: true
          description: null when the facet's subject is UNKNOWN / CONFLICT / NOT_RECOVERABLE.
    SellerCaseRow:
      type: object
      required: [case_id, workflow_type, state_raw, state_mapping, presence, customer_ref]
      additionalProperties: false
      properties:
        case_id: {$ref: './components/hr.yaml#/NativeID'}
        workflow_type: {$ref: './components/sales.yaml#/WorkflowType'}
        state_raw: {$ref: './components/hr.yaml#/RawText'}
        state_mapping: {$ref: './components/sales.yaml#/StateMapping'}
        presence: {$ref: './components/hr.yaml#/Presence'}
        lead: {$ref: './components/sales.yaml#/LeadFacts'}
        channel_raw: {$ref: './components/hr.yaml#/RawText'}
        customer_ref: {$ref: './components/billing.yaml#/BillingLink'}
        contact_outcome_current: {$ref: './components/writes.yaml#/ContactOutcomeCurrent'}
    SellerCaseListEnvelope:
      type: object
      required: [outcome, reasons, correlation_id, scope]
      additionalProperties: false
      properties:
        outcome: {$ref: './components/schemas.yaml#/Outcome'}
        reasons: {$ref: './components/hr.yaml#/Reasons'}
        correlation_id: {$ref: './components/schemas.yaml#/CorrelationID'}
        scope: {$ref: '#/components/schemas/ScopeEvidence'}
        count: {$ref: '#/components/schemas/ScopedCount'}
        items:
          type: array
          maxItems: 200
          items: {$ref: '#/components/schemas/SellerCaseRow'}
        page: {$ref: './components/schemas.yaml#/PageInfo'}
    SellerInvoiceUnit:
      type: object
      required: [unit, class]
      additionalProperties: false
      properties:
        unit: {$ref: './components/schemas.yaml#/Unit'}
        declared: {$ref: './components/billing.yaml#/BillingAmount'}
        payment_sum: {$ref: './components/billing.yaml#/BillingSum'}
        class: {$ref: './components/billing.yaml#/TotalClass'}
    SellerInvoiceRow:
      type: object
      required: [invoice_id, code_raw, lifecycle_state, case_ref, customer_ref, seller_is_caller, issuer_is_caller,
        items_source_complete, stages_source_complete, unknown_state_count, issued_at_raw]
      additionalProperties: false
      properties:
        invoice_id: {$ref: './components/hr.yaml#/NativeID'}
        code_raw: {$ref: './components/hr.yaml#/RawText'}
        lifecycle_state: {$ref: './components/billing.yaml#/InvoiceLifecycle'}
        case_ref: {$ref: './components/billing.yaml#/BillingLink'}
        customer_ref: {$ref: './components/billing.yaml#/BillingLink'}
        seller_is_caller: {type: boolean}
        issuer_is_caller: {type: boolean}
        items_source_complete: {type: boolean}
        stages_source_complete: {type: boolean}
        unknown_state_count: {type: integer, format: int64, minimum: 0}
        issued_at_raw: {$ref: './components/hr.yaml#/RawText'}
        units:
          type: array
          maxItems: 8
          items: {$ref: '#/components/schemas/SellerInvoiceUnit'}
    SellerInvoiceListEnvelope:
      type: object
      required: [outcome, reasons, correlation_id, scope]
      additionalProperties: false
      properties:
        outcome: {$ref: './components/schemas.yaml#/Outcome'}
        reasons: {$ref: './components/hr.yaml#/Reasons'}
        correlation_id: {$ref: './components/schemas.yaml#/CorrelationID'}
        scope: {$ref: '#/components/schemas/ScopeEvidence'}
        count: {$ref: '#/components/schemas/ScopedCount'}
        items:
          type: array
          maxItems: 200
          items: {$ref: '#/components/schemas/SellerInvoiceRow'}
        page: {$ref: './components/schemas.yaml#/PageInfo'}
```

Notes on the fragment:
- `seller_is_caller` / `issuer_is_caller` are both `true` under the recommended initial basis; they are kept so the same DTO works if M2 rec. 2 selects a single-column basis.
- No `confirmed` / `payable` field: "financially confirmed" has no Finance rule (GAP-03, D-25, D-43). The owner-only submission context (`/payments/invoices/{invoice_id}/submission`) remains the only source of "current payable".
- No `allowed_actions` on Seller reads until a Seller write is approved.

## 4. Registry entries the implementing PR must add (shape only)

```yaml
- operation_id: listMyCases
  method: GET
  path: /api/v1/me/cases
  category: OWN_READ
  capability: cases.own.read
  feature_flag: sales.read_shadow
  scope: OWN:custody_current_holder_user_id
  subject: AUTHENTICATED_USER_ID
  unproven_user: ROW_SCOPE_DENIED
  nt07: REQUIRED
```

The closed `OWN:<basis>` list in `operation_registry_test.go` gains a basis only in the PR that implements the approved one.

## 5. Validation performed on this proposal

- The YAML block in §3 parses (checked with a YAML parser in the RM-E04C QA run; see PR description).
- Every `$ref` target into `./components/...` exists at `c0d7f4b` (checked by the same QA script).
- None of the proposed paths or operationIds exists in `api/openapi/openapi.yaml` (enforced by `docs/governance/rm_e04c_test.go`).
