// Package custody names the custody module (Architecture V1 §5, §10): the
// append-only holding record of work items. In Slice 6 it is read/shadow
// only: CustodyEvents are imported from legacy evidence and never changed;
// the Assignment row is a derived, reconcilable projection of those events
// next to the legacy current-holder evidence. No Allocate, Transfer, Return
// or Release command exists. Current Custody, Original Owner, Next Actor and
// Credit Owner never collapse (GATE-0 INV-004).
package custody

import "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"

// Module is the custody write owner: schema "custody", NOLOGIN owner role
// "crm_owner_custody". No other module may write its schema.
const Module owner.Module = "custody"
