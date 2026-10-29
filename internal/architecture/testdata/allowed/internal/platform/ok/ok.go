// Fixture: platform may import kernel and platform packages; plain DML is fine. Never compiled.
package ok

import (
	"example.test/mod/internal/kernel/refs"
	"example.test/mod/internal/platform/db"
)

var q = "INSERT INTO syn.widget (label) VALUES ($1)"
var _, _ = refs.Domains, db.Open
