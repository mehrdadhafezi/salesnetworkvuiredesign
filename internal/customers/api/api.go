// Package api is the published surface of the customers module: the shadow
// import command contract. Records carry raw legacy evidence; the customers
// module never sees legacy table names (they stay in
// internal/integration/legacy). Customers are minted only by this command
// (one per issued-to subject); there is no registration, merge, resolve or
// canonicalisation command anywhere in Slice 5 (D-30 open).
package api

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
)

// ErrInvalidImport reports an import command that breaks its contract.
var ErrInvalidImport = errors.New("customers: invalid import")

var (
	keyPattern  = regexp.MustCompile(`^[1-9][0-9]{0,19}$`)
	uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	specPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*\.[a-z][a-z0-9_]*\.v[1-9][0-9]*$`)
	sysPattern  = regexp.MustCompile(`^[a-z][a-z0-9_]{1,15}$`)
)

// SourceRef is the provenance of one imported legacy row. The legacy key is
// provenance, never the target identity.
type SourceRef struct {
	System    string
	Spec      string
	Key       string
	BatchID   string
	RowSHA256 []byte
}

// Validate checks the reference shape.
func (s SourceRef) Validate() error {
	switch {
	case !sysPattern.MatchString(s.System):
		return fmt.Errorf("%w: source system %q", ErrInvalidImport, s.System)
	case !specPattern.MatchString(s.Spec):
		return fmt.Errorf("%w: source spec %q", ErrInvalidImport, s.Spec)
	case !keyPattern.MatchString(s.Key):
		return fmt.Errorf("%w: source key %q", ErrInvalidImport, s.Key)
	case !uuidPattern.MatchString(s.BatchID):
		return fmt.Errorf("%w: batch id %q", ErrInvalidImport, s.BatchID)
	case len(s.RowSHA256) != 32:
		return fmt.Errorf("%w: row checksum length %d", ErrInvalidImport, len(s.RowSHA256))
	}
	return nil
}

// SubjectKind is the kind of source record a customer subject stands for.
type SubjectKind string

const (
	// IssuedTo: one legacy invoice issued-to record (its own subject).
	IssuedTo SubjectKind = "INVOICE_ISSUED_TO"
	// SyntheticSubject: test-only explicit subjects (never from legacy).
	SyntheticSubject SubjectKind = "SYNTHETIC_SUBJECT"
)

// ContactKind / ContactRole.
type (
	ContactKind string
	ContactRole string
)

const (
	Phone     ContactKind = "PHONE"
	Email     ContactKind = "EMAIL"
	Primary   ContactRole = "PRIMARY"
	Secondary ContactRole = "SECONDARY"
)

// ContactRecord is one contact value of a subject, raw. ValidFrom/ValidTo
// are an explicit ownership window only when the source states one; an
// issued-to record only observes the value at ObservedRaw.
type ContactRecord struct {
	Kind         ContactKind
	Role         ContactRole
	Raw          string
	ValidFromRaw *string
	ValidToRaw   *string
}

// Scheme is an identifier scheme.
type Scheme string

const (
	// LegacyWPAccount is the legacy customer account id. In 2.0.160 it is
	// derived from the normalized phone, so it is never exclusive identity.
	LegacyWPAccount Scheme = "LEGACY_WP_ACCOUNT"
	WooCustomer     Scheme = "WOO_CUSTOMER"
	Biawin          Scheme = "BIAWIN"
	// Overlay (2.0.182) identifiers: modelled, not ingested in Slice 5.
	LegacyAggregateV1 Scheme = "LEGACY_AGGREGATE_V1"
	LegacyMapV2       Scheme = "LEGACY_MAP_V2"
	CRMCode           Scheme = "CRM_CODE"
)

// Exclusive reports whether one value of the scheme may belong to one
// customer only: sharing it across customers is a CONFLICT. Phone-derived
// schemes are never exclusive (their sharing is weak relation evidence).
func (s Scheme) Exclusive() bool {
	switch s {
	case WooCustomer, Biawin, CRMCode:
		return true
	}
	return false
}

func (s Scheme) valid() bool {
	switch s {
	case LegacyWPAccount, WooCustomer, Biawin, LegacyAggregateV1, LegacyMapV2, CRMCode:
		return true
	}
	return false
}

// Derivation states how an identifier value was produced.
type Derivation string

const (
	Direct       Derivation = "DIRECT"
	PhoneDerived Derivation = "PHONE_DERIVED"
)

// IdentifierRecord is one identifier claimed by a subject.
type IdentifierRecord struct {
	Scheme     Scheme
	Value      string
	Derivation Derivation
}

// SubjectRecord is one source subject (e.g. one invoice issued-to record).
type SubjectRecord struct {
	Kind        SubjectKind
	Source      SourceRef
	Name        *string
	ObservedRaw *string
	// Recoverable is false when the source record exists but its identity
	// payload is permanently missing (NOT_RECOVERABLE).
	Recoverable bool
	Contacts    []ContactRecord
	Identifiers []IdentifierRecord
}

// EvidenceKind is a SUPPLIED evidence kind (derived kinds are computed by
// the module). No legacy source supplies any of them in Slice 5.
type EvidenceKind string

const (
	// OTPContactControl: a subject proved control of a contact value at a
	// time (MEDIUM). It never proves two subjects are one customer.
	OTPContactControl EvidenceKind = "OTP_CONTACT_CONTROL"
	// VerifiedEquality / VerifiedDistinct: explicit STRONG evidence that
	// subjects are / are not the same customer. Synthetic in Slice 5.
	VerifiedEquality EvidenceKind = "VERIFIED_EQUALITY"
	VerifiedDistinct EvidenceKind = "VERIFIED_DISTINCT"
)

// EvidenceRecord is one supplied evidence row about one or more subjects
// (by their source keys within this command).
type EvidenceRecord struct {
	Kind        EvidenceKind
	Source      SourceRef
	SubjectKeys []string
	// ContactRaw names the contact for OTP evidence.
	ContactRaw  *string
	ObservedRaw *string
}

// CustomerImport is one customers import command for one staged batch.
type CustomerImport struct {
	BatchID      string
	SnapshotID   string
	SourceSystem string
	// Complete marks the command as covering every subject of SourceSystem:
	// only then may subjects missing from it be marked ABSENT_IN_LATEST
	// (never deleted). An incomplete command never marks anything absent.
	Complete bool
	Subjects []SubjectRecord
	Evidence []EvidenceRecord
}

// Validate checks the command shape.
func (in CustomerImport) Validate() error {
	if !uuidPattern.MatchString(in.BatchID) || in.SnapshotID == "" || !sysPattern.MatchString(in.SourceSystem) {
		return fmt.Errorf("%w: batch/snapshot/source system", ErrInvalidImport)
	}
	keys := map[string]bool{}
	for _, s := range in.Subjects {
		if err := s.Source.Validate(); err != nil {
			return err
		}
		if s.Source.BatchID != in.BatchID || s.Source.System != in.SourceSystem {
			return fmt.Errorf("%w: subject from another batch or system", ErrInvalidImport)
		}
		if s.Kind != IssuedTo && s.Kind != SyntheticSubject {
			return fmt.Errorf("%w: subject kind %q", ErrInvalidImport, s.Kind)
		}
		k := string(s.Kind) + "/" + s.Source.Key
		if keys[k] {
			return fmt.Errorf("%w: duplicate subject %s", ErrInvalidImport, s.Source.Key)
		}
		keys[k] = true
		for _, c := range s.Contacts {
			if (c.Kind != Phone && c.Kind != Email) || (c.Role != Primary && c.Role != Secondary) || len(c.Raw) > 200 {
				return fmt.Errorf("%w: contact of %s", ErrInvalidImport, s.Source.Key)
			}
		}
		for _, id := range s.Identifiers {
			if !id.Scheme.valid() || id.Value == "" || len(id.Value) > 120 || (id.Derivation != Direct && id.Derivation != PhoneDerived) {
				return fmt.Errorf("%w: identifier of %s", ErrInvalidImport, s.Source.Key)
			}
			if id.Scheme.Exclusive() && id.Derivation == PhoneDerived {
				return fmt.Errorf("%w: exclusive scheme %s cannot be phone-derived", ErrInvalidImport, id.Scheme)
			}
		}
	}
	for _, e := range in.Evidence {
		if err := e.Source.Validate(); err != nil {
			return err
		}
		switch e.Kind {
		case OTPContactControl:
			if len(e.SubjectKeys) != 1 || e.ContactRaw == nil {
				return fmt.Errorf("%w: OTP evidence names one subject and one contact", ErrInvalidImport)
			}
		case VerifiedEquality, VerifiedDistinct:
			if len(e.SubjectKeys) < 2 {
				return fmt.Errorf("%w: %s needs two or more subjects", ErrInvalidImport, e.Kind)
			}
		default:
			return fmt.Errorf("%w: evidence kind %q cannot be supplied", ErrInvalidImport, e.Kind)
		}
	}
	return nil
}

// ImportReport is the command result.
type ImportReport struct {
	Outcome result.Status
	Replay  bool
	RunID   string
	Counts  map[string]int64
}

// Importer is the customers import command (customers/imports).
type Importer interface {
	ImportCustomers(ctx context.Context, in CustomerImport) (ImportReport, error)
}

// Reader is the published read surface used by reconciliation.
type Reader interface {
	ResolutionReport(ctx context.Context) (Resolution, error)
	// SubjectProvenance returns, per source key, the stored provenance of
	// the customer standing for that subject (absent keys have no customer).
	SubjectProvenance(ctx context.Context, system string, kind SubjectKind, keys []string) (map[string]Provenance, error)
}

// Provenance is the stored source provenance of one customer.
type Provenance struct {
	BatchID   string // last batch that carried the subject
	RowSHA256 []byte
	Present   bool
}

// Resolution is the machine-readable resolution projection summary.
type Resolution struct {
	Customers                int64            `json:"customers"`
	CustomersPresent         int64            `json:"customers_present"`
	SubjectsPresent          int64            `json:"source_subjects_present"`
	ByState                  map[string]int64 `json:"customers_by_state"`
	CasesByState             map[string]int64 `json:"merge_cases_by_state"`
	CasesByReason            map[string]int64 `json:"merge_cases_by_reason"`
	PhoneOnlyCandidates      int64            `json:"phone_only_candidate_count"`
	OverlappingPhoneConflict int64            `json:"overlapping_phone_conflict_count"`
	DuplicateExternalID      int64            `json:"duplicate_external_id_conflict_count"`
	StrongEvidence           int64            `json:"strong_evidence_count"`
	WeakOnlyCustomers        int64            `json:"weak_only_customer_count"`
	Unresolved               int64            `json:"unresolved_count"`
	PhoneOnlyResolved        int64            `json:"phone_only_resolved_count"`
	PhoneOnlyMerges          int64            `json:"phone_only_merge_count"`
	Merges                   int64            `json:"merge_count"`
}
