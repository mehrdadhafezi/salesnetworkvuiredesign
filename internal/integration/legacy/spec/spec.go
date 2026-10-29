// Package spec defines versioned, immutable legacy extract specs
// (Architecture V1 §24.1). A spec names one legacy source table of one
// official legacy version, its columns in a fixed order, the source key and
// ordering, a per-column PII classification and the evidence in plugin/current
// that proves each column exists. Specs carry no transformation logic: staging
// is raw and verbatim (§24.3); target semantics belong to later slices.
//
// A published spec never changes. A semantic or shape change requires a new
// spec version (a new ID and a new staging table), never an edit of v1. The
// golden file in testdata freezes every published spec byte-for-byte.
package spec

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Errors returned by validation and lookup.
var (
	ErrInvalidSpec        = errors.New("spec: invalid extract spec")
	ErrDuplicateSpec      = errors.New("spec: duplicate spec id")
	ErrUnknownSpec        = errors.New("spec: unknown spec id")
	ErrUnsupportedVersion = errors.New("spec: unsupported legacy version")
)

// LegacyApplication identifies the official legacy application.
const LegacyApplication = "sales-network"

// OfficialLegacyVersion is the official legacy baseline (E0 §4B; Architecture
// V1 §2). Behavioural overlays (e.g. 2.0.182) are never extract sources.
const OfficialLegacyVersion = "2.0.160"

// ChecksumStrategy names the content-checksum definition every v1 spec uses
// (see package snapshot).
const ChecksumStrategy = "crm-legacy-content-v1/sha256"

// Nullability is the documented expectation for a source column. It is an
// expectation backed by evidence, never a staging rule: a violating value is
// preserved and reported, not rejected or fixed.
type Nullability string

const (
	NotNull  Nullability = "NOT_NULL"
	Nullable Nullability = "NULLABLE"
	// NullUnknown: the evidence does not settle nullability (e.g. the column
	// is declared differently on different install paths, or the table is a
	// WordPress core table not defined in plugin/current).
	NullUnknown Nullability = "UNKNOWN"
)

func (n Nullability) valid() bool { return n == NotNull || n == Nullable || n == NullUnknown }

// Classification is a minimal technical PII marker. It records what a column
// may contain; it decides no retention, masking, anonymisation, export or
// residency policy (owner decision D-23 / TAD (E0) #18).
type Classification string

const (
	ClassNone      Classification = "NONE"
	ClassPersonal  Classification = "PERSONAL"
	ClassSensitive Classification = "SENSITIVE"
)

func (c Classification) rank() int {
	switch c {
	case ClassNone:
		return 0
	case ClassPersonal:
		return 1
	case ClassSensitive:
		return 2
	}
	return -1
}

// KeyKind is how a source key value is checked and ordered.
type KeyKind string

// KeyUnsignedDecimal: the key is a canonical unsigned decimal string
// ("0" or no leading zero). Rows are strictly ascending by numeric value,
// which also proves key uniqueness. The raw text is never rewritten.
const KeyUnsignedDecimal KeyKind = "UNSIGNED_DECIMAL"

// Evidence locates a fact in the official legacy snapshot (plugin/current).
type Evidence struct {
	File  string `json:"file"`
	Lines string `json:"lines"`
	Note  string `json:"note"`
}

// Column is one source column in its fixed extract position.
type Column struct {
	// Name is the exact legacy source column name.
	Name string `json:"name"`
	// RawType is the legacy declared type where verified, else "UNKNOWN".
	RawType        string         `json:"raw_type"`
	Nullability    Nullability    `json:"nullability"`
	Classification Classification `json:"classification"`
	Evidence       string         `json:"evidence"`
	Note           string         `json:"note,omitempty"`
}

// StagingColumn is the raw text column that holds this source column.
func (c Column) StagingColumn() string { return "raw_" + strings.ToLower(c.Name) }

// Spec is one immutable, versioned extract spec.
type Spec struct {
	ID                string `json:"id"`
	Entity            string `json:"entity"`
	Version           int    `json:"version"`
	LegacyApplication string `json:"legacy_application"`
	LegacyVersion     string `json:"legacy_version"`
	// SourceTable is the unprefixed table name. The physical name is the
	// site's WordPress table prefix followed by SourceTable.
	SourceTable  string     `json:"source_table"`
	Columns      []Column   `json:"columns"`
	KeyColumns   []string   `json:"key_columns"`
	KeyKind      KeyKind    `json:"key_kind"`
	RowOrder     string     `json:"row_order"`
	Checksum     string     `json:"checksum"`
	StagingTable string     `json:"staging_table"`
	Evidence     []Evidence `json:"evidence"`
	Notes        []string   `json:"notes"`
}

var (
	idPattern      = regexp.MustCompile(`^legacy\.([a-z][a-z0-9_]{1,48})\.v([1-9][0-9]{0,2})$`)
	tablePattern   = regexp.MustCompile(`^[a-z][a-z0-9_]{1,62}$`)
	columnPattern  = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{0,58}$`)
	stagingPattern = regexp.MustCompile(`^stg_[a-z][a-z0-9_]{1,50}_v[1-9][0-9]{0,2}$`)
)

// Classification returns the highest classification of any column.
func (s Spec) Classification() Classification {
	best := ClassNone
	for _, c := range s.Columns {
		if c.Classification.rank() > best.rank() {
			best = c.Classification
		}
	}
	return best
}

// ColumnNames returns the source column names in extract order.
func (s Spec) ColumnNames() []string {
	out := make([]string, len(s.Columns))
	for i, c := range s.Columns {
		out[i] = c.Name
	}
	return out
}

// KeyIndexes returns the positions of the key columns in extract order.
func (s Spec) KeyIndexes() []int {
	out := make([]int, 0, len(s.KeyColumns))
	for _, k := range s.KeyColumns {
		for i, c := range s.Columns {
			if c.Name == k {
				out = append(out, i)
			}
		}
	}
	return out
}

// Validate checks every structural rule of a spec. Identifiers are restricted
// to safe patterns because staging SQL is generated only from validated specs.
func (s Spec) Validate() error {
	fail := func(format string, a ...any) error {
		return fmt.Errorf("%w: %s: %s", ErrInvalidSpec, s.ID, fmt.Sprintf(format, a...))
	}
	m := idPattern.FindStringSubmatch(s.ID)
	if m == nil {
		return fail("id must match legacy.<entity>.v<N>")
	}
	if m[1] != s.Entity || fmt.Sprint(s.Version) != m[2] {
		return fail("id does not match entity %q / version %d", s.Entity, s.Version)
	}
	if s.LegacyApplication != LegacyApplication {
		return fail("legacy application must be %q", LegacyApplication)
	}
	if s.LegacyVersion == "" {
		return fail("legacy version is required")
	}
	if !tablePattern.MatchString(s.SourceTable) {
		return fail("source table %q is not a safe identifier", s.SourceTable)
	}
	if want := fmt.Sprintf("stg_%s_v%d", s.Entity, s.Version); s.StagingTable != want || !stagingPattern.MatchString(s.StagingTable) {
		return fail("staging table must be %q", want)
	}
	if len(s.Columns) == 0 || len(s.Columns) > 200 {
		return fail("column count %d out of range", len(s.Columns))
	}
	seen := map[string]bool{}
	for _, c := range s.Columns {
		if !columnPattern.MatchString(c.Name) {
			return fail("column %q is not a safe identifier", c.Name)
		}
		lower := strings.ToLower(c.Name)
		if seen[lower] {
			return fail("duplicate column %q (case-insensitive)", c.Name)
		}
		// Staging columns are raw_<lowercase name>, so a source column can
		// never collide with a provenance column; case-insensitive
		// uniqueness above keeps staging names unique.
		seen[lower] = true
		if c.RawType == "" || !c.Nullability.valid() || c.Classification.rank() < 0 || c.Evidence == "" {
			return fail("column %q needs raw type, nullability, classification and evidence", c.Name)
		}
	}
	if len(s.KeyColumns) != 1 {
		return fail("v1 specs require exactly one key column")
	}
	if !seen[strings.ToLower(s.KeyColumns[0])] || len(s.KeyIndexes()) != 1 {
		return fail("key column %q is not a declared column", s.KeyColumns[0])
	}
	if s.KeyKind != KeyUnsignedDecimal {
		return fail("unsupported key kind %q", s.KeyKind)
	}
	if s.RowOrder == "" || s.Checksum != ChecksumStrategy {
		return fail("row order and checksum strategy are required")
	}
	if len(s.Evidence) == 0 {
		return fail("table-level evidence is required")
	}
	return nil
}

// Registry is an immutable set of specs.
type Registry struct {
	byID map[string]Spec
	ids  []string
}

// NewRegistry validates every spec and rejects duplicate IDs, duplicate
// source tables within one legacy version and duplicate staging tables.
func NewRegistry(specs ...Spec) (*Registry, error) {
	r := &Registry{byID: map[string]Spec{}}
	tables := map[string]string{}
	staging := map[string]string{}
	for _, s := range specs {
		if err := s.Validate(); err != nil {
			return nil, err
		}
		if _, dup := r.byID[s.ID]; dup {
			return nil, fmt.Errorf("%w: %s", ErrDuplicateSpec, s.ID)
		}
		tk := s.LegacyVersion + "/" + s.SourceTable + "/v" + fmt.Sprint(s.Version)
		if prev, dup := tables[tk]; dup {
			return nil, fmt.Errorf("%w: %s and %s both extract %s", ErrDuplicateSpec, prev, s.ID, tk)
		}
		if prev, dup := staging[s.StagingTable]; dup {
			return nil, fmt.Errorf("%w: %s and %s share staging table %s", ErrDuplicateSpec, prev, s.ID, s.StagingTable)
		}
		tables[tk], staging[s.StagingTable] = s.ID, s.ID
		r.byID[s.ID] = s
		r.ids = append(r.ids, s.ID)
	}
	sort.Strings(r.ids)
	return r, nil
}

// Lookup returns the spec with the given ID.
func (r *Registry) Lookup(id string) (Spec, error) {
	s, ok := r.byID[id]
	if !ok {
		return Spec{}, fmt.Errorf("%w: %q", ErrUnknownSpec, id)
	}
	return s, nil
}

// ForVersion returns the spec only if it was published for legacyVersion; a
// spec of another version is never silently reused.
func (r *Registry) ForVersion(id, legacyVersion string) (Spec, error) {
	s, err := r.Lookup(id)
	if err != nil {
		return Spec{}, err
	}
	if s.LegacyVersion != legacyVersion {
		return Spec{}, fmt.Errorf("%w: %s is published for %s, not %q", ErrUnsupportedVersion, id, s.LegacyVersion, legacyVersion)
	}
	return s, nil
}

// SupportsVersion reports whether any spec is published for legacyVersion.
func (r *Registry) SupportsVersion(legacyVersion string) bool {
	for _, s := range r.byID {
		if s.LegacyVersion == legacyVersion {
			return true
		}
	}
	return false
}

// IDs returns every spec ID in sorted order.
func (r *Registry) IDs() []string { return append([]string(nil), r.ids...) }

// Specs returns every spec in ID order.
func (r *Registry) Specs() []Spec {
	out := make([]Spec, 0, len(r.ids))
	for _, id := range r.ids {
		out = append(out, r.byID[id])
	}
	return out
}
