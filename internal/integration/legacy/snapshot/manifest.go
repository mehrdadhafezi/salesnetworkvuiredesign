package snapshot

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/spec"
)

// Manifest format identity.
const (
	ManifestFile          = "manifest.json"
	TablesDir             = "tables"
	ManifestFormat        = "crm.legacy.snapshot"
	ManifestFormatVersion = 1
	ChecksumAlgorithm     = "sha256"
	maxManifestBytes      = 1 << 20
)

// DataOrigin declares where the package content comes from. It records the
// operator's statement; it is not an anonymisation or retention policy.
type DataOrigin string

const (
	// OriginSynthetic: obviously synthetic fixture data (the only origin CI uses).
	OriginSynthetic DataOrigin = "synthetic"
	// OriginUnclassified: anything else. Loading it requires an explicit
	// operator acknowledgement because TAD (E0) #18 / D-23 are unresolved.
	OriginUnclassified DataOrigin = "unclassified"
)

// Completeness is the extractor's statement about one entity's coverage.
type Completeness string

const (
	Complete            Completeness = "COMPLETE"
	Incomplete          Completeness = "INCOMPLETE"
	CompletenessUnknown Completeness = "UNKNOWN"
	NotRecoverable      Completeness = "NOT_RECOVERABLE"
)

func (c Completeness) valid() bool {
	return c == Complete || c == Incomplete || c == CompletenessUnknown || c == NotRecoverable
}

// Legacy identifies the legacy application and version of the source.
type Legacy struct {
	Application string `json:"application"`
	Version     string `json:"version"`
	Build       string `json:"build,omitempty"`
}

// Source identifies the extraction source without any secret.
type Source struct {
	// ID is a non-secret label such as "synthetic-fixture-01".
	ID string `json:"id"`
	// Timezone is the source database zone where known (IANA name). It is
	// recorded as evidence only; staging never reinterprets timestamps.
	Timezone string `json:"timezone,omitempty"`
	// TablePrefix is the site's WordPress table prefix, where known.
	TablePrefix string `json:"table_prefix,omitempty"`
}

// Tool identifies the extraction tool.
type Tool struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Entity is one table file of the package.
type Entity struct {
	SpecID        string       `json:"spec_id"`
	File          string       `json:"file"`
	Columns       []string     `json:"columns"`
	RowCount      int64        `json:"row_count"`
	ByteSize      int64        `json:"byte_size"`
	FileSHA256    string       `json:"file_sha256"`
	ContentSHA256 string       `json:"content_sha256"`
	Completeness  Completeness `json:"completeness"`
	Notes         []string     `json:"notes,omitempty"`
}

// Manifest describes a snapshot package. Unknown fields are rejected, so a
// DSN, password or key cannot be carried in it unnoticed.
type Manifest struct {
	Format            string     `json:"format"`
	FormatVersion     int        `json:"format_version"`
	SnapshotID        string     `json:"snapshot_id"`
	Legacy            Legacy     `json:"legacy"`
	Source            Source     `json:"source"`
	DataOrigin        DataOrigin `json:"data_origin"`
	ExtractedAt       string     `json:"extracted_at"`
	Tool              Tool       `json:"extraction_tool"`
	ChecksumAlgorithm string     `json:"checksum_algorithm"`
	Entities          []Entity   `json:"entities"`
	Warnings          []string   `json:"warnings"`
}

// Manifest validation errors.
var (
	ErrManifest           = errors.New("snapshot: invalid manifest")
	ErrUnsupportedVersion = errors.New("snapshot: unsupported legacy version")
	ErrUnsupportedSpec    = errors.New("snapshot: unsupported extract spec")
)

var (
	snapshotIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{7,127}$`)
	sourceIDPattern   = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{2,63}$`)
	prefixPattern     = regexp.MustCompile(`^[A-Za-z0-9_]{1,32}$`)
	zonePattern       = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_+\-]*(/[A-Za-z0-9_+\-]+){0,2}$`)
	toolPattern       = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+-]{0,63}$`)
	hexSHA256         = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// ParseManifest decodes manifest bytes strictly: unknown fields, trailing
// data and oversize input are rejected. It does not validate content.
func ParseManifest(b []byte) (Manifest, error) {
	if len(b) > maxManifestBytes {
		return Manifest{}, fmt.Errorf("%w: larger than %d bytes", ErrManifest, maxManifestBytes)
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	var m Manifest
	if err := dec.Decode(&m); err != nil {
		return Manifest{}, fmt.Errorf("%w: %v", ErrManifest, err)
	}
	if _, err := dec.Token(); err != io.EOF {
		return Manifest{}, fmt.Errorf("%w: trailing data after the manifest object", ErrManifest)
	}
	return m, nil
}

// EntityFile is the only accepted path of an entity's table file.
func EntityFile(specID string) string { return TablesDir + "/" + specID + ".ndjson" }

// Validate checks the manifest against the spec registry. A spec published
// for another legacy version is never used for this package.
func (m Manifest) Validate(reg *spec.Registry) error {
	bad := func(format string, a ...any) error {
		return fmt.Errorf("%w: "+format, append([]any{ErrManifest}, a...)...)
	}
	switch {
	case m.Format != ManifestFormat:
		return bad("format must be %q", ManifestFormat)
	case m.FormatVersion != ManifestFormatVersion:
		return bad("format_version must be %d", ManifestFormatVersion)
	case !snapshotIDPattern.MatchString(m.SnapshotID):
		return bad("snapshot_id must match %s", snapshotIDPattern)
	case m.Legacy.Application != spec.LegacyApplication:
		return bad("legacy.application must be %q", spec.LegacyApplication)
	case m.Legacy.Version == "":
		return bad("legacy.version is required")
	case m.Legacy.Build != "" && !toolPattern.MatchString(m.Legacy.Build):
		return bad("legacy.build is malformed")
	case !sourceIDPattern.MatchString(m.Source.ID):
		return bad("source.id must be a non-secret label matching %s", sourceIDPattern)
	case m.Source.Timezone != "" && !zonePattern.MatchString(m.Source.Timezone):
		return bad("source.timezone must be an IANA zone name")
	case m.Source.TablePrefix != "" && !prefixPattern.MatchString(m.Source.TablePrefix):
		return bad("source.table_prefix is malformed")
	case m.DataOrigin != OriginSynthetic && m.DataOrigin != OriginUnclassified:
		return bad("data_origin must be %q or %q", OriginSynthetic, OriginUnclassified)
	case !toolPattern.MatchString(m.Tool.Name) || !toolPattern.MatchString(m.Tool.Version):
		return bad("extraction_tool name and version are required")
	case m.ChecksumAlgorithm != ChecksumAlgorithm:
		return bad("checksum_algorithm must be %q", ChecksumAlgorithm)
	case len(m.Entities) == 0:
		return bad("at least one entity is required")
	case m.Warnings == nil:
		return bad("warnings must be present (use [] when there are none)")
	}
	if t, err := time.Parse(time.RFC3339Nano, m.ExtractedAt); err != nil || !strings.HasSuffix(m.ExtractedAt, "Z") || t.IsZero() {
		return bad("extracted_at must be an RFC 3339 UTC timestamp ending in Z")
	}
	if !reg.SupportsVersion(m.Legacy.Version) {
		return fmt.Errorf("%w: %q (no published specs; specs of another version are never reused)", ErrUnsupportedVersion, m.Legacy.Version)
	}
	seen := map[string]bool{}
	for i, e := range m.Entities {
		s, err := reg.ForVersion(e.SpecID, m.Legacy.Version)
		if err != nil {
			return fmt.Errorf("%w: entity %d: %v", ErrUnsupportedSpec, i, err)
		}
		if seen[e.SpecID] {
			return bad("entity %s declared twice", e.SpecID)
		}
		seen[e.SpecID] = true
		if e.File != EntityFile(e.SpecID) {
			return bad("entity %s file must be %q", e.SpecID, EntityFile(e.SpecID))
		}
		if !equalStrings(e.Columns, s.ColumnNames()) {
			return bad("entity %s columns must equal the spec columns in order", e.SpecID)
		}
		if e.RowCount < 0 || e.ByteSize < 0 {
			return bad("entity %s counts must be non-negative", e.SpecID)
		}
		if !hexSHA256.MatchString(e.FileSHA256) || !hexSHA256.MatchString(e.ContentSHA256) {
			return bad("entity %s checksums must be lowercase hex SHA-256", e.SpecID)
		}
		if !e.Completeness.valid() {
			return bad("entity %s completeness is invalid", e.SpecID)
		}
	}
	return nil
}

// ContentIdentity is the package content checksum: it covers the legacy
// version and, per entity in spec-ID order, the row count, content checksum
// and completeness. It excludes extracted_at, source labels, warnings and
// file names, so the same content always has the same identity.
func (m Manifest) ContentIdentity() string {
	ents := append([]Entity(nil), m.Entities...)
	sort.Slice(ents, func(i, j int) bool { return ents[i].SpecID < ents[j].SpecID })
	h := sha256.New()
	fmt.Fprintf(h, "crm-legacy-package-v1\n%s\n%s\n", m.Legacy.Application, m.Legacy.Version)
	for _, e := range ents {
		fmt.Fprintf(h, "%s %d %s %s\n", e.SpecID, e.RowCount, e.ContentSHA256, e.Completeness)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
