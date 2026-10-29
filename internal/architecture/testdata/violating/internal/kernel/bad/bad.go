// Fixture: deliberately violates every kernel rule. Never compiled.
package bad

import (
	"math/rand"
	"net/http"
	"os"
	"time"

	"example.test/mod/internal/billing"
	"example.test/mod/internal/kernel/refs"
)

func F(x float64, t time.Time) {
	_ = time.Now()
	_ = float32(1)
	_ = "sn_invoices"
	_ = t.Local()
	_, _ = refs.NewLegacyRef("wp", "users", "1")
	_, _, _, _ = rand.Int(), http.MethodGet, os.Args, billing.X
}
