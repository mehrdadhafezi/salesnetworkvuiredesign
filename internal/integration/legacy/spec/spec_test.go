package spec

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the published-spec golden file")

const golden = "testdata/specs_v1_2.0.160.golden.json"

func render(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(Default().Specs()); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// TestPublishedSpecsAreImmutable freezes every published v1 spec byte for
// byte. Changing a v1 spec fails here: publish a new version instead.
func TestPublishedSpecsAreImmutable(t *testing.T) {
	got := render(t)
	if *update {
		if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("published v1 specs changed; v1 is immutable — add a v2 spec instead (or regenerate the golden only for a not-yet-published spec)")
	}
	sum := sha256.Sum256(want)
	t.Logf("published spec set sha256 %s", hex.EncodeToString(sum[:]))
}

func TestDefaultRegistry(t *testing.T) {
	r := Default()
	want := []string{
		"legacy.hr_assignment.v1", "legacy.hr_assignment_history.v1", "legacy.hr_department.v1", "legacy.hr_level.v1",
		"legacy.hr_position.v1", "legacy.hr_position_role_mapping.v1", "legacy.hr_profile.v1", "legacy.hr_team.v1",
		"legacy.hr_unit.v1", "legacy.wp_user.v1", "legacy.wp_usermeta.v1",
	}
	if got := r.IDs(); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("ids = %v", got)
	}
	versioned := regexp.MustCompile(`^legacy\.[a-z_]+\.v1$`)
	for _, s := range r.Specs() {
		if !versioned.MatchString(s.ID) || s.LegacyVersion != OfficialLegacyVersion || s.Checksum != ChecksumStrategy {
			t.Errorf("%s: id/version/checksum = %s %s %s", s.ID, s.ID, s.LegacyVersion, s.Checksum)
		}
		if s.KeyColumns[0] != s.Columns[0].Name {
			t.Errorf("%s: key column must be the first column", s.ID)
		}
		for _, c := range s.Columns {
			if !strings.Contains(c.Evidence, "plugin/current/") {
				t.Errorf("%s.%s: evidence must cite plugin/current", s.ID, c.Name)
			}
		}
	}
	if !r.SupportsVersion("2.0.160") || r.SupportsVersion("2.0.182") || r.SupportsVersion("2.0.156") {
		t.Fatal("only 2.0.160 is supported")
	}
}

func TestNoCredentialOrOverlaySources(t *testing.T) {
	r := Default()
	for _, s := range r.Specs() {
		for _, c := range s.Columns {
			if strings.Contains(strings.ToLower(c.Name), "pass") || c.Name == "user_activation_key" {
				t.Errorf("%s extracts credential column %s", s.ID, c.Name)
			}
		}
		// 2.0.182-only structures are behavioural overlay, never v1 sources.
		if strings.Contains(s.SourceTable, "aggregate_customer") || strings.Contains(s.SourceTable, "customer_id_map") {
			t.Errorf("%s stages a 2.0.182-only table %s", s.ID, s.SourceTable)
		}
	}
	u, err := r.Lookup("legacy.wp_user.v1")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(u.ColumnNames(), ",") != "ID,user_login,user_email,display_name,user_registered" {
		t.Fatalf("wp_user columns = %v", u.ColumnNames())
	}
	if u.Classification() != ClassPersonal {
		t.Fatalf("wp_user classification = %s", u.Classification())
	}
	p, _ := r.Lookup("legacy.hr_profile.v1")
	if p.Classification() != ClassSensitive || len(p.Columns) != 26 {
		t.Fatalf("hr_profile classification %s, %d columns", p.Classification(), len(p.Columns))
	}
	if pos, _ := r.Lookup("legacy.hr_position.v1"); pos.Classification() != ClassNone {
		t.Fatalf("hr_position classification = %s", pos.Classification())
	}
}

// TestWpUsermetaExtractionScopeIsUnresolved guards the deferred wp_usermeta
// meta_key decision: real extraction stays blocked until an Owner-approved
// allowlist exists. Removing this entry requires that decision.
func TestWpUsermetaExtractionScopeIsUnresolved(t *testing.T) {
	reason, blocked := ExtractionScopeUnresolved("legacy.wp_usermeta.v1")
	if !blocked || !strings.Contains(reason, "meta_key") || !strings.Contains(reason, "allowlist") {
		t.Fatalf("wp_usermeta extraction scope must stay unresolved: %q %v", reason, blocked)
	}
	s, err := Default().Lookup("legacy.wp_usermeta.v1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(s.Notes, " "), "v1 defines no filter") {
		t.Fatalf("wp_usermeta spec note changed: %v", s.Notes)
	}
	for _, id := range Default().IDs() {
		if _, b := ExtractionScopeUnresolved(id); b && id != "legacy.wp_usermeta.v1" {
			t.Errorf("unexpected scope block on %s", id)
		}
	}
	if _, b := ExtractionScopeUnresolved("legacy.nope.v1"); b {
		t.Fatal("unknown spec must not be listed")
	}
}

func TestStagingColumnsDeterministic(t *testing.T) {
	u, _ := Default().Lookup("legacy.wp_user.v1")
	got := []string{}
	for _, c := range u.Columns {
		got = append(got, c.StagingColumn())
	}
	if strings.Join(got, ",") != "raw_id,raw_user_login,raw_user_email,raw_display_name,raw_user_registered" {
		t.Fatalf("staging columns = %v", got)
	}
}

func valid() Spec { return hrPositionRoleMappingV1() }

func TestRegistryRejectsDuplicates(t *testing.T) {
	if _, err := NewRegistry(valid(), valid()); !errors.Is(err, ErrDuplicateSpec) {
		t.Fatalf("duplicate id: %v", err)
	}
	other := valid()
	other.ID, other.Entity = "legacy.other_mapping.v1", "other_mapping"
	other.StagingTable = "stg_other_mapping_v1"
	if _, err := NewRegistry(valid(), other); !errors.Is(err, ErrDuplicateSpec) {
		t.Fatalf("duplicate source table: %v", err)
	}
}

func TestValidateRejectsUnsafeOrIncompleteSpecs(t *testing.T) {
	cases := map[string]func(*Spec){
		"unversioned id":     func(s *Spec) { s.ID = "legacy.hr_position_role_mapping" },
		"id/entity mismatch": func(s *Spec) { s.Entity = "other" },
		"injected table":     func(s *Spec) { s.SourceTable = `sn_x"; DROP TABLE y; --` },
		"injected column":    func(s *Spec) { s.Columns[1].Name = "legacy_role; DELETE" },
		"quoted column":      func(s *Spec) { s.Columns[1].Name = `a"b` },
		"case duplicate":     func(s *Spec) { s.Columns[1].Name = "ID" },
		"bad staging table":  func(s *Spec) { s.StagingTable = "stg_other_v1" },
		"missing evidence":   func(s *Spec) { s.Columns[2].Evidence = "" },
		"bad nullability":    func(s *Spec) { s.Columns[2].Nullability = "MAYBE" },
		"bad classification": func(s *Spec) { s.Columns[2].Classification = "SECRET" },
		"unknown key column": func(s *Spec) { s.KeyColumns = []string{"nope"} },
		"two key columns":    func(s *Spec) { s.KeyColumns = []string{"id", "position_id"} },
		"bad key kind":       func(s *Spec) { s.KeyKind = "STRING" },
		"no checksum":        func(s *Spec) { s.Checksum = "" },
		"wrong application":  func(s *Spec) { s.LegacyApplication = "other" },
		"no table evidence":  func(s *Spec) { s.Evidence = nil },
		"no columns":         func(s *Spec) { s.Columns = nil },
	}
	for name, mutate := range cases {
		s := valid()
		s.Columns = append([]Column(nil), s.Columns...)
		mutate(&s)
		if err := s.Validate(); !errors.Is(err, ErrInvalidSpec) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
	if err := valid().Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestForVersionNeverReusesAnotherVersion(t *testing.T) {
	r := Default()
	if _, err := r.ForVersion("legacy.hr_profile.v1", "2.0.182"); !errors.Is(err, ErrUnsupportedVersion) {
		t.Fatalf("version mismatch: %v", err)
	}
	if _, err := r.ForVersion("legacy.nope.v1", "2.0.160"); !errors.Is(err, ErrUnknownSpec) {
		t.Fatalf("unknown spec: %v", err)
	}
	if _, err := r.ForVersion("legacy.hr_profile.v1", "2.0.160"); err != nil {
		t.Fatal(err)
	}
}
