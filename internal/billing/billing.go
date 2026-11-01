// Package billing names the billing module (Architecture V1 §5, §10): the
// Invoice, its InvoiceItems (issuance snapshots) and PaymentStages. In
// Slice 7 it is a read/shadow model only: no IssueInvoice, IssueNextStage or
// CancelInvoice exists. Invoice, stage and payment states stay separate; the
// canonical invoice-total (TAD #19) and main-status (TAD #20) resolvers are
// deferred, so no canonical total or status is ever claimed.
package billing

import "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"

// Module is the billing write owner: schema "billing", NOLOGIN owner role
// "crm_owner_billing". No other module may write its schema.
const Module owner.Module = "billing"
