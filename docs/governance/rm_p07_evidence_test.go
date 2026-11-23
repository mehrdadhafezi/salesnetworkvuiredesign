package governance

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// TestRMP07EvidenceIsFinanceOnly pins docs/evidence/rm-p07 to the RM-P07
// Finance workspace screenshots. Other browser suites write their evidence
// through the generic E2E_SCREENSHOT_DIR; the Finance suite writes only
// "finance-" files through E2E_FINANCE_SCREENSHOT_DIR. A stray image from
// another slice (or a missing Finance view) fails governance.
func TestRMP07EvidenceIsFinanceOnly(t *testing.T) {
	dir := filepath.Join("..", "evidence", "rm-p07")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var want []string
	for _, view := range []string{"landing", "queue", "exceptions", "receipts", "denied"} {
		for _, vp := range []string{"desktop", "tablet", "mobile"} {
			want = append(want, "finance-"+view+"-"+vp+".png")
		}
	}
	var got []string
	for _, e := range entries {
		got = append(got, e.Name())
	}
	slices.Sort(want)
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Fatalf("docs/evidence/rm-p07 must hold exactly the Finance workspace screenshots\n got: %v\nwant: %v", got, want)
	}
}
