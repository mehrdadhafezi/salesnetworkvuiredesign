// Package payments names the payments module (Architecture V1 §5, §10):
// PaymentAttempts, GatewayAttempts and Receipt references. In Slice 7 it is
// a read/shadow model only: no payment submission, gateway initiation or
// callback processing, Finance approval, receipt upload/finalisation or
// posting exists. Legacy 'approved' and 'reconcile_error' are preserved raw;
// gateway auto-approval is gateway_auto provenance, never Finance review
// (D-25 open).
package payments

import "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"

// Module is the payments write owner: schema "payments", NOLOGIN owner role
// "crm_owner_payments". No other module may write its schema.
const Module owner.Module = "payments"
