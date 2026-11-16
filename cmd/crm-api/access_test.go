package main

import (
	"strings"
	"testing"

	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/rollout"
)

// guardFlags is every flag a composed guard is constructed with (the New…API
// functions of this package). Availability may only come from these.
var guardFlags = map[string]bool{
	HRReadFlag: true, CustomersFlag: true, SalesFlag: true, BillingFlag: true, ReportingFlag: true, FinanceFlag: true, BiawinFlag: true,
	rollout.WriteFlag(ContactOutcomeFamily): true, rollout.WriteFlag(HRRequestFamily): true, WritebackReadFlag: true,
	rollout.WriteFlag(PaymentsFamily): true, FinanceReviewFlag: true, CommissionFlag: true,
	OperationsReadFlag: true, rollout.WriteFlag(WorkFamily): true, ProviderReconcileFlag: true,
}

// L (registry side) / RM-ACCESS-01: every registered capability has exactly
// one availability flag, and nothing else does.
func TestCapabilityFlagsCoverRegistry(t *testing.T) {
	if len(identityapi.Capabilities()) != 43 {
		t.Fatalf("registry has %d capabilities; update RM-ACCESS-01 evidence and capabilityFlags", len(identityapi.Capabilities()))
	}
	for _, c := range identityapi.Capabilities() {
		k, ok := capabilityFlags[c.Key]
		if !ok {
			t.Errorf("capability %s has no availability flag (it would be BLOCKED forever)", c.Key)
			continue
		}
		if !guardFlags[k] {
			t.Errorf("capability %s mapped to %q, which no guard uses", c.Key, k)
		}
	}
	if len(capabilityFlags) != len(identityapi.Capabilities()) {
		t.Fatalf("capabilityFlags has %d entries for %d capabilities", len(capabilityFlags), len(identityapi.Capabilities()))
	}
}

// DEF-ACCESS-01: the HR read flag governs exactly the six HR read
// capabilities, and no other module's capability depends on it.
func TestHRFlagGovernsOnlyHRRead(t *testing.T) {
	hr := map[identityapi.Capability]bool{}
	for _, c := range identityapi.HRReadCapabilities() {
		hr[c.Key] = true
	}
	for c, k := range capabilityFlags {
		if (k == HRReadFlag) != hr[c] {
			t.Errorf("capability %s: flag %s (HR read set member: %v)", c, k, hr[c])
		}
	}
}

// O. Availability never comes from a write-back switch (real legacy
// write-back stays separately gated and is not implemented), and the HR
// request family's availability is its command flag, never a final-apply
// switch.
func TestAvailabilityNeverFromWritebackOrApply(t *testing.T) {
	for c, k := range capabilityFlags {
		if strings.HasSuffix(k, "_writeback") || strings.Contains(k, "apply") || strings.Contains(k, "post") || strings.Contains(k, "activation") {
			t.Errorf("capability %s availability uses %q", c, k)
		}
	}
}

func TestCapabilityAvailabilityNeedsFlagStore(t *testing.T) {
	if _, err := capabilityAvailability(nil); err == nil {
		t.Fatal("availability without a flag store")
	}
}
