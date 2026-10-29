// Fixture: one module reaching into another module's store. Never compiled.
package sales

import "example.test/mod/internal/billing/store"

var _ = store.X
