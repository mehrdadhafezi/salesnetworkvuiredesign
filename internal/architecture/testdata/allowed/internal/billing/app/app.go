// Fixture: a module may import its own store. Never compiled.
package app

import (
	"example.test/mod/internal/billing/store"
	"example.test/mod/internal/kernel/refs"
)

var _, _ = store.X, refs.Domains
