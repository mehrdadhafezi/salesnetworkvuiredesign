// Package clock provides the runtime implementation of the kernel
// timeutil.Clock contract. The kernel never reads the system clock; this
// adapter, outside the kernel, is the only place that does. Tests use
// timeutil.FixedClock.
package clock

import (
	"time"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
)

// System is the real UTC clock.
type System struct{}

var _ timeutil.Clock = System{}

// Now returns the current instant in UTC.
func (System) Now() timeutil.Instant {
	// time.Now is never the zero time, so NewInstant cannot fail here.
	i, _ := timeutil.NewInstant(time.Now())
	return i
}
