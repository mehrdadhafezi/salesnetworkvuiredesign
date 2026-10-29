// Fixture: legacy knowledge is allowed only here. Never compiled.
package legacy

import "example.test/mod/internal/kernel/refs"

var table = "sn_invoices"

func Ref() (refs.LegacyRef, error) { return refs.NewLegacyRef("wp", "users", "1") }
