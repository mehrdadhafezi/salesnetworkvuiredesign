package shadow_test

import (
	"crypto/sha256"
	"testing"

	casesapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/shadow"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/staging"
)

func row(key string, cols map[string]string) shadow.Row {
	raw := map[string]*string{}
	for k, v := range cols {
		v := v
		raw[k] = &v
	}
	sum := sha256.Sum256([]byte(key))
	return shadow.Row{Key: key, SHA: sum[:], Raw: raw}
}

func batchWithLogs(completeness string, logs ...shadow.Row) *shadow.Batch {
	return &shadow.Batch{BatchID: "11111111-1111-4111-8111-111111111111", SnapshotID: "syn", Entities: map[string]shadow.Entity{
		shadow.SpecDistItem: {SpecID: shadow.SpecDistItem, State: staging.EntityLoaded, Completeness: "COMPLETE",
			Rows: []shadow.Row{row("501", map[string]string{"status": "delivered_to_seller", "current_owner_user_id": "5"})}},
		shadow.SpecDistItemLog: {SpecID: shadow.SpecDistItemLog, State: staging.EntityLoaded, Completeness: completeness, Rows: logs},
	}}
}

func logRow(key, action, to, pos string) shadow.Row {
	return row(key, map[string]string{"item_id": "501", "action": action, "to_user_id": to, "to_position": pos, "created_at": "2025-02-01 08:00:00"})
}

func facets(t *testing.T, b *shadow.Batch) map[casesapi.Facet]casesapi.Resolution {
	t.Helper()
	in, _ := shadow.CasesCommand(b, nil)
	out := map[casesapi.Facet]casesapi.Resolution{}
	for _, a := range in.Attributions {
		out[a.Facet] = a.Resolution
	}
	return out
}

// Original owner/seller resolve only from COMPLETE logs with no unmapped
// action before the chosen row; otherwise an earlier assignment could be
// missing or hidden, so the facet is UNKNOWN (never a later holder).
func TestOriginAttributionNeedsCompleteEvidence(t *testing.T) {
	logs := []shadow.Row{logRow("1", "item_created_from_mis_pool", "4", "supervisor"), logRow("2", "delivered_to_seller", "5", "seller")}
	if f := facets(t, batchWithLogs("COMPLETE", logs...)); f[casesapi.OriginalOwner] != casesapi.Resolved || f[casesapi.OriginalSeller] != casesapi.Resolved {
		t.Fatalf("complete logs: %v", f)
	}
	if f := facets(t, batchWithLogs("INCOMPLETE", logs...)); f[casesapi.OriginalOwner] != casesapi.Unknown || f[casesapi.OriginalSeller] != casesapi.Unknown {
		t.Fatalf("incomplete logs must not resolve origin: %v", f)
	}
	hidden := append([]shadow.Row{logRow("0", "synthetic_unverified_action", "3", "supervisor")}, logs...)
	if f := facets(t, batchWithLogs("COMPLETE", hidden...)); f[casesapi.OriginalOwner] != casesapi.Unknown {
		t.Fatalf("unmapped action before the first assignment: %v", f)
	}
	if f := facets(t, batchWithLogs("COMPLETE")); f[casesapi.OriginalOwner] != casesapi.NotRecoverable {
		t.Fatalf("complete logs without any assignment: %v", f)
	}
	if f := facets(t, batchWithLogs("INCOMPLETE")); f[casesapi.OriginalOwner] != casesapi.Unknown {
		t.Fatalf("incomplete logs without any assignment must be UNKNOWN: %v", f)
	}
}
