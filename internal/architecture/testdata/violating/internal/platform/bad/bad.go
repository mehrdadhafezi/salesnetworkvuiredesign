// Fixture: platform code that imports a business module and the test-only
// harness, and runs DDL at runtime. Never compiled.
package bad

import (
	"example.test/mod/internal/billing"
	"example.test/mod/internal/platform/pgtest"
)

var ddl = "CREATE TABLE platform_x.thing (id int)"

var _, _ = billing.X, pgtest.New
