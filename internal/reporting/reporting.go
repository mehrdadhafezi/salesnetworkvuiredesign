// Package reporting names the reporting module (Architecture V1 §20): read
// projections built offline from owner published APIs, versioned metric
// contracts, and the Invoice Review QUERY SURFACE (no export, no target UI:
// D-23, D-24). Reporting tables are never source of truth; they never read
// or write another module's schema. MIS reports are reporting projections:
// there is no MIS-owned data in Slice 8, so no mis module exists yet.
package reporting

import "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"

// Module is the reporting write owner: schema "reporting", NOLOGIN owner role
// "crm_owner_reporting". Only the offline projection rebuild writes it.
const Module owner.Module = "reporting"
