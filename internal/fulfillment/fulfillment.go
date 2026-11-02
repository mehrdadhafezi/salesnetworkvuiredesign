// Package fulfillment names the fulfillment module (Architecture V1 §13):
// operational execution — FulfillmentCase, FulfillmentAssignment,
// ServiceEvidence, ServiceOutcome. Slice 10 implements only the READ /
// SHADOW subset: an offline import of normalized legacy evidence into
// insert-only, run-pinned shadow tables and a GET-only read surface. No
// assignment, completion, shipping, upgrade or evidence-upload command
// exists. A fulfillment completion is never an invoice payment, a
// subscription activation, a sales completion or a finance settlement.
package fulfillment

import "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"

// Module is the fulfillment write owner: schema "fulfillment", NOLOGIN owner
// role "crm_owner_fulfillment". Only the offline shadow import writes it.
const Module owner.Module = "fulfillment"
