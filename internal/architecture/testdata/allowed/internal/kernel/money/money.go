// Fixture: allowed kernel production code. Never compiled.
package money

import (
	"encoding/json"
	"time"

	"example.test/mod/internal/kernel/refs"
)

func Add(a, b int64) int64 { return a + b }

var _ = json.Marshal
var _ = time.UTC
var _ = refs.Domains
