// Fixture: violates the Slice 10 fulfillment read-path rules. Never compiled.
package query

// read-path-import: a read never imports.
import _ "example.test/mod/internal/fulfillment/imports"

// biawin-read-write + append-only-write: a read never completes a case.
const complete = "UPDATE fulfillment.fulfillment_case SET status_mapped = 'COMPLETED'"

// biawin-forbidden-command: no assignment, shipping, evidence upload or
// completion route.
func AssignExpert()   {}
func ShipOrder()      {}
func UploadEvidence() {}

const route = "/api/v1/fulfillment/cases/{case_id}/complete"
