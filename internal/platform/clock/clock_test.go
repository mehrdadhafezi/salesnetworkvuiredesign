package clock

import (
	"testing"
	"time"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
)

func TestSystemClockIsUTC(t *testing.T) {
	var c timeutil.Clock = System{}
	before := time.Now().UTC().Add(-time.Second)
	now := c.Now()
	if now.IsZero() || now.Time().Location() != time.UTC {
		t.Fatalf("Now = %v", now)
	}
	if now.Time().Before(before) || now.Time().After(time.Now().UTC().Add(time.Second)) {
		t.Fatalf("Now %v is not current", now)
	}
}

func TestFixedClockForTests(t *testing.T) {
	at, _ := timeutil.ParseRFC3339("2026-10-29T00:00:00Z")
	var c timeutil.Clock = timeutil.FixedClock{At: at}
	if !c.Now().Equal(at) {
		t.Fatal("fixed clock must be deterministic")
	}
}
