// Package commission names the commission module (Architecture V1 §19).
// Slice 13 implements only the controlled, ZERO-FINANCIAL-EFFECT subset: a
// versioned policy model observed from legacy evidence, a pure deterministic
// calculation registry, immutable attribution / input snapshots, an offline
// shadow run that recalculates historical entitlements from their event-time
// snapshots and records parity against the legacy engines, reconciliation
// findings and a GET-only read surface. There is no approval, posting,
// crediting, settlement or reversal command: OPD-09 and D-36 are unresolved
// and D-41 keeps both engines unpublished. Commission never writes finance,
// wallet, billing, payments, organization or any other module's tables.
package commission

import "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"

// Module is the commission write owner: schema "commission", NOLOGIN owner
// role "crm_owner_commission". Only the offline shadow job writes it.
const Module owner.Module = "commission"
