// Fixture: violates the Slice 14 catalog rules. Never compiled.
package catalog

// module-boundary: the catalog depends on no business module.
import _ "example.test/mod/internal/biawin/api"

// foreign-schema-write + append-only-write: only internal/fulfillment/work
// may update the work case current state.
const mutateWork = "UPDATE fulfillment.work_case SET state = 'COMPLETED'"
