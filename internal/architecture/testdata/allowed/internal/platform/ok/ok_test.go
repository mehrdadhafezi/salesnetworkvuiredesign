// Fixture: tests may import pgtest and use DDL. Never compiled.
package ok

import (
	"testing"

	"example.test/mod/internal/platform/pgtest"
)

func TestX(t *testing.T) {
	_ = pgtest.New
	_ = "DROP TABLE fixture"
}
