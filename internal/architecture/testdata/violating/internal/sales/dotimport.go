// Fixture: a dot import hides the factory call from the legacyref-factory rule. Never compiled.
package sales

import . "example.test/mod/internal/kernel/refs"

func Ref() (LegacyRef, error) { return NewLegacyRef("wp", "users", "1") }
