// Package legacytest builds SYNTHETIC legacy snapshot packages for tests.
// It is test-only infrastructure (imported from _test.go files only). Every
// value is obviously fake: example.invalid addresses, "TEST-PHONE-…"
// placeholders that are not phone numbers, "Dev User"/"Synthetic Employee"
// names. No production, plugin-example or real-looking personal data.
package legacytest

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/snapshot"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/spec"
)

// T and N build cells.
func T(s string) snapshot.Cell { return snapshot.Text(s) }

// N is a NULL cell.
func N() snapshot.Cell { return snapshot.Null() }

// Header returns a synthetic manifest header.
func Header(snapshotID, extractedAt string) snapshot.Header {
	return snapshot.Header{
		SnapshotID: snapshotID,
		Legacy:     snapshot.Legacy{Application: spec.LegacyApplication, Version: spec.OfficialLegacyVersion, Build: "2026-10-10-finance-dialog-layout"},
		Source:     snapshot.Source{ID: "synthetic-fixture-01", Timezone: "Asia/Tehran", TablePrefix: "synth_"},
		DataOrigin: snapshot.OriginSynthetic, ExtractedAt: extractedAt,
		Tool: snapshot.Tool{Name: "legacytest", Version: "1.0.0"}, Warnings: []string{},
	}
}

// Rows returns synthetic rows for every published v1 spec, exercising NULL
// versus empty string, Unicode, raw serialized/JSON meta text, raw legacy
// DATETIME text (including a zero date) and unfamiliar status strings.
func Rows() []snapshot.EntityRows {
	reg := spec.Default()
	out := make([]snapshot.EntityRows, 0, len(reg.IDs()))
	for _, s := range reg.Specs() {
		var rows [][]snapshot.Cell
		for i := 1; i <= 3; i++ {
			rows = append(rows, row(s, i))
		}
		out = append(out, snapshot.EntityRows{SpecID: s.ID, Rows: rows, Completeness: snapshot.Complete})
	}
	return out
}

func row(s spec.Spec, i int) []snapshot.Cell {
	cells := make([]snapshot.Cell, len(s.Columns))
	for c, col := range s.Columns {
		switch {
		case c == s.KeyIndexes()[0]:
			cells[c] = T(fmt.Sprint(i * 10)) // 10, 20, 30: ascending, gaps preserved
		case i == 2 && col.Nullability != spec.NotNull:
			cells[c] = N()
		case i == 3:
			cells[c] = T("") // empty string, distinct from NULL
		default:
			cells[c] = T(value(s.Entity, col.Name, i))
		}
	}
	return cells
}

func value(entity, column string, i int) string {
	switch column {
	case "user_login":
		return fmt.Sprintf("dev-user-%d", i)
	case "user_email":
		return fmt.Sprintf("dev.user.%d@example.invalid", i)
	case "display_name", "hr_display_name":
		return fmt.Sprintf("Dev User %d — کاربر آزمایشی", i)
	case "meta_value":
		return `a:1:{s:13:"administrator";b:1;}` // raw serialized text, never parsed
	case "context":
		return `{"synthetic":true,"note":"خط\nدوم","list":[1,2]}`
	case "work_phone", "emergency_phone":
		return fmt.Sprintf("TEST-PHONE-%04d", i)
	case "national_id":
		return fmt.Sprintf("SYNTHETIC-NID-%d", i)
	case "created_at", "updated_at", "user_registered", "effective_from":
		if i == 1 {
			return "0000-00-00 00:00:00" // raw legacy zero date; never fixed
		}
		return "2026-01-02 03:04:05"
	case "employment_status", "relationship_type", "action":
		return "synthetic_unfamiliar_status_x" // unknown business strings survive
	}
	return fmt.Sprintf("syn-%s-%s-%d", entity, column, i)
}

// Write builds a complete synthetic package under t.TempDir() and returns
// its directory.
func Write(t testing.TB, snapshotID, extractedAt string, entities []snapshot.EntityRows) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "snapshot")
	if _, err := snapshot.Write(dir, spec.Default(), Header(snapshotID, extractedAt), entities); err != nil {
		t.Fatal(err)
	}
	return dir
}
