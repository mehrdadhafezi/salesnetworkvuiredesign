// Fixture: Slice 14 fulfillment work rules. Never compiled.
package work

import (
	// module-boundary: work intake reads biawin only through biawin/api.
	_ "example.test/mod/internal/biawin/activation"
	// module-boundary: the catalog is read only through catalog/api.
	_ "example.test/mod/internal/catalog/registry"
	// allowed: published api packages.
	_ "example.test/mod/internal/biawin/api"
	_ "example.test/mod/internal/catalog/api"
)

// allowed: the single mutable Slice 14 table, from its owner package.
const updateCase = "UPDATE fulfillment.work_case SET state = $1, version = version + 1 WHERE case_id = $2 AND version = $3"

// append-only-write: the case is never deleted; history is never rewritten.
const deleteCase = "DELETE FROM fulfillment.work_case WHERE case_id = $1"
const rewriteHistory = "UPDATE fulfillment.work_event SET kind = 'X'"

// allowed: an exact Slice 14 command route.
const evidenceRoute = "/api/v1/fulfillment/work-cases/{case_id}/evidence-records"

// biawin-forbidden-command: any other evidence spelling stays forbidden.
const evidenceUpload = "/api/v1/fulfillment/work-cases/{case_id}/evidence"
