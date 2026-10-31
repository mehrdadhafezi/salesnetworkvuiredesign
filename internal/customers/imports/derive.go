package imports

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers/domain"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers/store"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
)

// derivedSpec names derived evidence provenance (computed by this module).
const derivedSpec = "customers.derivation.v1"

func uniqSorted(xs []string) []string {
	m := map[string]bool{}
	for _, x := range xs {
		m[x] = true
	}
	out := make([]string, 0, len(m))
	for x := range m {
		out = append(out, x)
	}
	sort.Strings(out)
	return out
}

func maxStrength(ev []domain.Evidence) string {
	best := domain.Weak
	for _, e := range ev {
		switch {
		case e.Strength == domain.Strong:
			return domain.Strong
		case e.Strength == domain.Medium:
			best = domain.Medium
		}
	}
	return best
}

// derive computes derived evidence and the candidate cases. Groups:
//   - SHARED_PHONE: present customers sharing a normalized phone (≥2);
//     PHONE_MATCH (weak discovery) plus, per pair with explicit windows,
//     OVERLAPPING_PHONE_OWNERSHIP (overlap → conflict) or
//     SEQUENTIAL_PHONE_REUSE (disjoint → distinct, no merge); supplied OTP
//     on that phone joins the case (medium relation, never equality);
//   - SHARED_ACCOUNT_REF / SHARED_EXCLUSIVE_IDENTIFIER: one identifier value
//     on ≥2 customers (exclusive scheme → contradiction);
//   - SUPPLIED_EVIDENCE: explicit equality/distinctness over its subjects;
//   - SOURCE_NOT_RECOVERABLE evidence for non-recoverable subjects.
//
// It returns the cases and the identifier ids in CONFLICT.
func derive(ctx context.Context, tx db.Tx, snap store.Snapshot, sup []supplied, in api.CustomerImport, now time.Time,
	bump, note func(string, int64)) ([]store.CaseInput, []string, error) {
	var seen []string
	record := func(kind, reference string, subjects []string, keyParts ...string) (string, domain.Evidence, error) {
		k := domain.Catalogue[kind]
		subjects = uniqSorted(subjects)
		parts := append([]string{"DERIVED", kind, strings.Join(subjects, ",")}, keyParts...)
		id, isNew, err := store.RecordEvidence(ctx, tx, store.EvidenceInput{Key: domain.Key(parts...), Kind: kind, Strength: k.Strength,
			Polarity: k.Polarity, Origin: "DERIVED", Reference: reference, System: in.SourceSystem, Spec: derivedSpec, Subjects: subjects},
			in.BatchID, now)
		if err == nil {
			seen = append(seen, id)
			if isNew {
				bump("evidence_derived_new", 1)
			}
		}
		return id, domain.Evidence{Kind: kind, Strength: k.Strength, Polarity: k.Polarity}, err
	}
	var cases []store.CaseInput
	addCase := func(reason string, members, evIDs []string, ev []domain.Evidence, keyParts ...string) {
		state, why := domain.Resolve(ev)
		cases = append(cases, store.CaseInput{Key: domain.Key(append([]string{"CASE", reason}, keyParts...)...), Reason: reason,
			State: state, StateReason: why, MaxStrength: maxStrength(ev), Members: uniqSorted(members), Evidence: uniqSorted(evIDs)})
	}

	// Shared phones.
	byPhone := map[string][]store.Contact{}
	for _, c := range snap.Contacts {
		if c.Kind == string(api.Phone) && c.Normalized != nil {
			byPhone[*c.Normalized] = append(byPhone[*c.Normalized], c)
		}
	}
	phones := make([]string, 0, len(byPhone))
	for p := range byPhone {
		phones = append(phones, p)
	}
	sort.Strings(phones)
	for _, phone := range phones {
		group := byPhone[phone]
		var members []string
		for _, c := range group {
			members = append(members, c.CustomerID)
		}
		members = uniqSorted(members)
		if len(members) < 2 {
			continue
		}
		var evIDs []string
		var ev []domain.Evidence
		id, e, err := record(domain.PhoneMatch, "equal normalized phone on different subjects (discovery only)", members, "phone", phone)
		if err != nil {
			return nil, nil, err
		}
		evIDs, ev = append(evIDs, id), append(ev, e)
		for i := 0; i < len(group); i++ {
			for j := i + 1; j < len(group); j++ {
				a, b := group[i], group[j]
				if a.CustomerID == b.CustomerID {
					continue
				}
				wa, wb := domain.ParseWindow(a.ValidFrom, a.ValidTo), domain.ParseWindow(b.ValidFrom, b.ValidTo)
				if !wa.Known || !wb.Known {
					continue // point observations: temporal evidence insufficient, nothing claimed
				}
				kind, ref := domain.SequentialPhoneReuse, "same phone in disjoint ownership windows (reuse; distinct subjects)"
				if domain.Overlap(wa, wb) {
					kind, ref = domain.OverlappingPhoneOwnership, "same phone in overlapping ownership windows (unexplained)"
				}
				id, e, err := record(kind, ref, []string{a.CustomerID, b.CustomerID}, "phone", phone, "pair", a.ID, b.ID)
				if err != nil {
					return nil, nil, err
				}
				evIDs, ev = append(evIDs, id), append(ev, e)
			}
		}
		for _, s := range sup {
			if s.rec.Kind == api.OTPContactControl && s.contact != nil && *s.contact == phone {
				k := domain.Catalogue[domain.OTPContactControl]
				evIDs = append(evIDs, s.id)
				ev = append(ev, domain.Evidence{Kind: domain.OTPContactControl, Strength: k.Strength, Polarity: k.Polarity})
			}
		}
		addCase("SHARED_PHONE", members, evIDs, ev, "phone", phone)
		note("phone_groups", 1)
	}

	// Shared identifiers.
	type idKey struct{ scheme, value string }
	byID := map[idKey][]store.Identifier{}
	for _, i := range snap.Identifiers {
		byID[idKey{i.Scheme, i.Value}] = append(byID[idKey{i.Scheme, i.Value}], i)
	}
	keys := make([]idKey, 0, len(byID))
	for k := range byID {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(a, b int) bool {
		if keys[a].scheme != keys[b].scheme {
			return keys[a].scheme < keys[b].scheme
		}
		return keys[a].value < keys[b].value
	})
	var conflicting []string
	for _, k := range keys {
		group := byID[k]
		var members []string
		for _, i := range group {
			members = append(members, i.CustomerID)
		}
		members = uniqSorted(members)
		if len(members) < 2 {
			continue
		}
		kind, reason, ref := domain.AccountRefMatch, "SHARED_ACCOUNT_REF", "same non-exclusive identifier (phone-derived legacy account); a link, never equality"
		if api.Scheme(k.scheme).Exclusive() {
			kind, reason, ref = domain.ExclusiveIdentifierShared, "SHARED_EXCLUSIVE_IDENTIFIER", "exclusive identifier claimed by different subjects"
			for _, i := range group {
				conflicting = append(conflicting, i.ID)
			}
		}
		id, e, err := record(kind, ref, members, "identifier", k.scheme, k.value)
		if err != nil {
			return nil, nil, err
		}
		addCase(reason, members, []string{id}, []domain.Evidence{e}, "identifier", k.scheme, k.value)
	}

	// Supplied equality / distinctness.
	for _, s := range sup {
		if s.rec.Kind != api.VerifiedEquality && s.rec.Kind != api.VerifiedDistinct {
			continue
		}
		k := domain.Catalogue[string(s.rec.Kind)]
		addCase("SUPPLIED_EVIDENCE", s.subjects, []string{s.id},
			[]domain.Evidence{{Kind: string(s.rec.Kind), Strength: k.Strength, Polarity: k.Polarity}},
			"supplied", s.rec.Source.Spec, s.rec.Source.Key)
	}

	// Non-recoverable subjects.
	var unrecoverable []string
	for id, rec := range snap.Customers {
		if !rec {
			unrecoverable = append(unrecoverable, id)
		}
	}
	sort.Strings(unrecoverable)
	for _, id := range unrecoverable {
		if _, _, err := record(domain.SourceNotRecoverable, "issued-to identity permanently missing in the source", []string{id}); err != nil {
			return nil, nil, err
		}
	}

	n, err := store.MarkAbsentEvidence(ctx, tx, "DERIVED", seen)
	if err != nil {
		return nil, nil, err
	}
	bump("evidence_derived_marked_absent", n)
	return cases, conflicting, nil
}
