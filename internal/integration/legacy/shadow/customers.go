package shadow

// Slice 5 customer identity shadow connector. It maps staged legacy invoice
// "issued-to" rows (legacy.invoice_issued_to.v1) into the neutral customers
// import command and reconciles the result. It knows legacy column names;
// the customers module never does.
//
// Every invoice row is one source subject and becomes one customer. Phone
// is discovery data only: it is carried as a contact point, never as a key.
// customer_wp_id is the WordPress account found or created by normalized
// phone in 2.0.160 (class-sn-plugin.php:5023-5048), so it is passed as a
// PHONE_DERIVED identifier, never as identity proof. No legacy column is
// STRONG evidence; none is invented here. sn_aggregate_customer_ids and
// sn_customer_id_map_v2 are not ingested (not an approved overlay source).

import (
	"context"
	"strings"

	customersapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
)

// SpecInvoiceIssuedTo is the staged spec Slice 5 reads.
const SpecInvoiceIssuedTo = "legacy.invoice_issued_to.v1"

// CustomerSpecs lists every staged spec Slice 5 reads.
func CustomerSpecs() []string { return []string{SpecInvoiceIssuedTo} }

// maxContact bounds a raw contact value accepted by the command.
const maxContact = 200

func trimmed(raw *string) string {
	if raw == nil {
		return ""
	}
	return strings.TrimSpace(*raw)
}

// CustomersCommand builds the customers import command from staged rows.
// The mapping counts report skipped or degraded values; nothing is dropped
// silently.
func CustomersCommand(b *Batch) (customersapi.CustomerImport, map[string]int64) {
	counts := map[string]int64{}
	e := b.Entity(SpecInvoiceIssuedTo)
	in := customersapi.CustomerImport{BatchID: b.BatchID, SnapshotID: b.SnapshotID, SourceSystem: SourceSystem, Complete: e.Complete()}
	for _, r := range e.Rows {
		s := customersapi.SubjectRecord{Kind: customersapi.IssuedTo,
			Source:      customersapi.SourceRef{System: SourceSystem, Spec: SpecInvoiceIssuedTo, Key: r.Key, BatchID: b.BatchID, RowSHA256: r.SHA},
			ObservedRaw: text(r.Get("created_at"))}
		if n := trimmed(r.Get("customer_name")); n != "" {
			s.Name = &n
		}
		for _, c := range []struct {
			col  string
			role customersapi.ContactRole
		}{{"customer_phone", customersapi.Primary}, {"customer_phone_secondary", customersapi.Secondary}} {
			v := trimmed(r.Get(c.col))
			switch {
			case v == "":
				counts[c.col+"_empty"]++
			case len(v) > maxContact:
				counts[c.col+"_too_long"]++
			default:
				s.Contacts = append(s.Contacts, customersapi.ContactRecord{Kind: customersapi.Phone, Role: c.role, Raw: v})
			}
		}
		switch k, bad := key(r.Get("customer_wp_id")); {
		case k != nil:
			s.Identifiers = append(s.Identifiers, customersapi.IdentifierRecord{Scheme: customersapi.LegacyWPAccount, Value: *k, Derivation: customersapi.PhoneDerived})
			counts["phone_derived_account_refs"]++
		case bad:
			counts["customer_wp_id_unreadable"]++
		default:
			counts["customer_wp_id_absent"]++
		}
		// A subject with neither a name nor a usable contact cannot be
		// recovered from this source: it stays a customer, marked
		// NOT_RECOVERABLE, never dropped or zero-filled.
		s.Recoverable = s.Name != nil || len(s.Contacts) > 0
		if !s.Recoverable {
			counts["subjects_not_recoverable"]++
		}
		in.Subjects = append(in.Subjects, s)
	}
	return in, counts
}

// CustomerReport is the Slice 5 resolution report (JSON).
type CustomerReport struct {
	Command       string                  `json:"command"`
	SnapshotID    string                  `json:"snapshot_id"`
	BatchID       string                  `json:"batch_id"`
	Outcome       result.Status           `json:"outcome"`
	Reasons       []reasoncodes.Code      `json:"reasons"`
	ConsumedSpecs []string                `json:"consumed_specs"`
	Staged        map[string]int64        `json:"staged"`
	Resolution    customersapi.Resolution `json:"resolution"`
	// Invariants are merge-blocking; any false value fails the command.
	Invariants map[string]bool   `json:"invariants"`
	Coverage   map[string]string `json:"coverage"`
	Notes      []string          `json:"notes"`
}

// ReconcileCustomers reports the customer identity resolution against the
// staged batch. It reads only.
func ReconcileCustomers(ctx context.Context, b *Batch, reader customersapi.Reader) (CustomerReport, error) {
	e := b.Entity(SpecInvoiceIssuedTo)
	rep := CustomerReport{Command: "reconcile-customers", SnapshotID: b.SnapshotID, BatchID: b.BatchID, Reasons: []reasoncodes.Code{},
		ConsumedSpecs: CustomerSpecs(), Staged: map[string]int64{"invoice_issued_to_rows": int64(len(e.Rows))},
		Coverage: map[string]string{
			SpecInvoiceIssuedTo:         coverage(e),
			"woo_customers":             "UNKNOWN (not staged in Slice 5)",
			"sn_aggregate_customer_ids": "NOT_INGESTED (not an approved overlay evidence source)",
			"sn_customer_id_map_v2":     "NOT_INGESTED (not an approved overlay evidence source)",
			"strong_identity_evidence":  "NONE IN LEGACY SOURCE (only explicitly supplied evidence can be STRONG)",
		},
		Notes: []string{"one source subject = one customer; there is no merge, resolve or canonicalisation path",
			"phone is discovery data only; customer_wp_id is phone-derived (2.0.160 class-sn-plugin.php:5023-5048) and is WEAK relation evidence",
			"contact values are masked in every read surface; staging rows are never exposed"}}
	res, err := reader.ResolutionReport(ctx)
	if err != nil {
		return rep, err
	}
	rep.Resolution = res
	rep.Invariants = map[string]bool{
		"phone_only_resolved_count_is_zero": res.PhoneOnlyResolved == 0,
		"phone_only_merge_count_is_zero":    res.PhoneOnlyMerges == 0,
		"merge_count_is_zero":               res.Merges == 0,
	}
	violated := false
	for _, ok := range rep.Invariants {
		violated = violated || !ok
	}
	switch {
	case violated:
		rep.Outcome = result.Conflict
		rep.Reasons = append(rep.Reasons, reasoncodes.LinkageConflict)
	case !e.Usable():
		rep.Outcome = result.Unknown
		rep.Reasons = append(rep.Reasons, reasoncodes.SourceUnavailable)
	case !e.Complete() || res.SubjectsPresent != int64(len(e.Rows)):
		rep.Outcome = result.Partial
		rep.Reasons = append(rep.Reasons, reasoncodes.CoverageIncomplete)
		rep.Notes = append(rep.Notes, "staged rows and present source subjects differ, or the staged entity is not COMPLETE")
	default:
		rep.Outcome = result.Success
	}
	return rep, nil
}

func coverage(e Entity) string {
	switch {
	case !e.Usable():
		return "UNKNOWN (not staged)"
	case !e.Complete():
		return "PARTIAL (staged, not COMPLETE)"
	}
	return "COMPLETE"
}
