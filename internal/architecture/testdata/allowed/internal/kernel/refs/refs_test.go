// Fixture: kernel tests may use math/rand, time/tzdata and the factory. Never compiled.
package refs

import (
	"math/rand"
	"testing"
	"time"
	_ "time/tzdata"
)

func TestX(t *testing.T) {
	_ = rand.Int()
	_ = time.Local
	_, _ = NewLegacyRef("wp", "users", "1")
	_ = t
}
