package snapshot_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/legacytest"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/snapshot"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/spec"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
)

const t0 = "2026-10-29T08:00:00Z"

func reg() *spec.Registry { return spec.Default() }

// --- canonical row encoding ---------------------------------------------------

func TestRawFidelityRoundTrip(t *testing.T) {
	cells := []snapshot.Cell{
		snapshot.Null(), snapshot.Text(""), snapshot.Text("null"),
		snapshot.Text("کارمند آزمایشی — Synthetic Employee"),
		snapshot.Text(`a:1:{s:13:"administrator";b:1;}`),
		snapshot.Text("{\"k\":[1,2],\"n\":null}\n\ttab\\slash\r\x01\x1f\x7f"),
		snapshot.Text("0000-00-00 00:00:00"), snapshot.Text(" padded "), snapshot.Text(" "),
	}
	line, err := snapshot.EncodeRow(cells)
	if err != nil {
		t.Fatal(err)
	}
	got, err := snapshot.DecodeRow(line, len(cells))
	if err != nil {
		t.Fatal(err)
	}
	for i := range cells {
		if got[i] != cells[i] {
			t.Fatalf("cell %d: got %+v want %+v", i, got[i], cells[i])
		}
	}
	if !got[0].Null || got[1].Null || got[1].Text != "" || got[2].Null {
		t.Fatal("NULL, empty string and the text \"null\" must stay distinct")
	}
	if !bytes.Contains(line, []byte(`\u0001`)) || !bytes.Contains(line, []byte("کارمند")) || bytes.Contains(line, []byte(`\u06`)) {
		t.Fatalf("encoding not canonical: %s", line)
	}
}

func TestDecodeRejectsNonCanonicalOrUntypedRows(t *testing.T) {
	cases := map[string]string{
		"whitespace":        `[ "a"]`,
		"escaped non-ascii": `["\u0627"]`,
		"escaped slash":     `["a\/b"]`,
		"number cell":       `[1]`,
		"bool cell":         `[true]`,
		"object cell":       `[{"a":"b"}]`,
		"nested array":      `[["a"]]`,
		"not array":         `{"a":"b"}`,
		"top null":          `null`,
		"empty line":        ``,
		"uppercase hex":     `["\u001F"]`,
	}
	for name, line := range cases {
		if _, err := snapshot.DecodeRow([]byte(line), 1); err == nil {
			t.Errorf("%s: accepted %q", name, line)
		}
	}
	if _, err := snapshot.DecodeRow([]byte(`["a","b"]`), 1); !errors.Is(err, snapshot.ErrColumnCount) {
		t.Errorf("width: %v", err)
	}
	if _, err := snapshot.DecodeRow([]byte("[\"\xff\"]"), 1); err == nil {
		t.Error("invalid UTF-8 accepted")
	}
	if _, err := snapshot.DecodeRow([]byte(`["a\u0000b"]`), 1); !errors.Is(err, snapshot.ErrNULCharacter) {
		t.Errorf("NUL: %v", err)
	}
	if _, err := snapshot.EncodeRow([]snapshot.Cell{snapshot.Text("\xff")}); !errors.Is(err, snapshot.ErrInvalidUTF8) {
		t.Errorf("encode invalid UTF-8: %v", err)
	}
}

// --- package build, determinism and checksums ---------------------------------

func TestPackageValidatesAndIsDeterministic(t *testing.T) {
	a := legacytest.Write(t, "synthetic-snapshot-0001", t0, legacytest.Rows())
	b := legacytest.Write(t, "synthetic-snapshot-0001", "2027-01-01T00:00:00Z", legacytest.Rows())
	pa, err := snapshot.Open(a, reg())
	if err != nil {
		t.Fatal(err)
	}
	pb, err := snapshot.Open(b, reg())
	if err != nil {
		t.Fatal(err)
	}
	reps, err := pa.Validate()
	if err != nil {
		t.Fatal(err)
	}
	if len(reps) != len(reg().IDs()) {
		t.Fatalf("validated %d entities", len(reps))
	}
	for _, r := range reps {
		if r.Status != result.Success || r.ParsedRows != 3 || r.ActualFileSHA256 != r.ExpectedFileSHA256 || r.ActualContentSHA != r.ExpectedContentSHA {
			t.Fatalf("%s: %+v", r.SpecID, r)
		}
	}
	// Content checksums do not depend on extraction time; the manifest does.
	if pa.Manifest.ContentIdentity() != pb.Manifest.ContentIdentity() {
		t.Fatal("content identity must not depend on extracted_at")
	}
	if pa.ManifestSHA256 == pb.ManifestSHA256 {
		t.Fatal("manifest checksums must differ when extracted_at differs")
	}
	for i := range pa.Manifest.Entities {
		ea, eb := pa.Manifest.Entities[i], pb.Manifest.Entities[i]
		if ea.FileSHA256 != eb.FileSHA256 || ea.ContentSHA256 != eb.ContentSHA256 {
			t.Fatalf("%s checksums are not deterministic", ea.SpecID)
		}
		fa, _ := os.ReadFile(filepath.Join(a, ea.File))
		fb, _ := os.ReadFile(filepath.Join(b, eb.File))
		if !bytes.Equal(fa, fb) {
			t.Fatalf("%s table bytes are not deterministic", ea.SpecID)
		}
	}
	// A golden content checksum pins the checksum definition itself.
	h := snapshot.NewContentHasher("legacy.example.v1")
	h.AddRow([]byte(`["1",null,""]`))
	if got := h.Sum(); got != goldenContent {
		t.Fatalf("content checksum definition changed: %s", got)
	}
}

// goldenContent is SHA-256("crm-legacy-content-v1\nlegacy.example.v1\n13:[\"1\",null,\"\"]\n").
const goldenContent = "98a8e59dc1e8b91ef50dd03f58346eeb921b1c9c8f9a340f04301ce4a2a78ab6"

func writeOne(t *testing.T, id string, rows [][]snapshot.Cell, c snapshot.Completeness) string {
	t.Helper()
	return legacytest.Write(t, "synthetic-snapshot-0002", t0, []snapshot.EntityRows{{SpecID: id, Rows: rows, Completeness: c}})
}

func mapping(id string) []snapshot.Cell {
	return []snapshot.Cell{snapshot.Text(id), snapshot.Text("synthetic_role"), snapshot.Text("1"), snapshot.Null(), snapshot.Text("1"),
		snapshot.Text("2026-01-02 03:04:05"), snapshot.Text("2026-01-02 03:04:05")}
}

const mappingID = "legacy.hr_position_role_mapping.v1"

func editManifest(t *testing.T, dir string, edit func(map[string]any)) {
	t.Helper()
	p := filepath.Join(dir, snapshot.ManifestFile)
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	edit(m)
	out, _ := json.MarshalIndent(m, "", "  ")
	if err := os.WriteFile(p, out, 0o600); err != nil {
		t.Fatal(err)
	}
}

func entity(m map[string]any) map[string]any { return m["entities"].([]any)[0].(map[string]any) }

func TestManifestValidation(t *testing.T) {
	good := writeOne(t, mappingID, [][]snapshot.Cell{mapping("1")}, snapshot.Complete)
	if _, err := snapshot.Open(good, reg()); err != nil {
		t.Fatalf("valid manifest rejected: %v", err)
	}
	cases := map[string]struct {
		edit func(map[string]any)
		want error
	}{
		"missing snapshot_id":    {func(m map[string]any) { delete(m, "snapshot_id") }, snapshot.ErrManifest},
		"missing legacy":         {func(m map[string]any) { delete(m, "legacy") }, snapshot.ErrManifest},
		"missing source id":      {func(m map[string]any) { m["source"] = map[string]any{} }, snapshot.ErrManifest},
		"missing extracted_at":   {func(m map[string]any) { delete(m, "extracted_at") }, snapshot.ErrManifest},
		"missing warnings":       {func(m map[string]any) { delete(m, "warnings") }, snapshot.ErrManifest},
		"missing tool":           {func(m map[string]any) { delete(m, "extraction_tool") }, snapshot.ErrManifest},
		"missing entities":       {func(m map[string]any) { m["entities"] = []any{} }, snapshot.ErrManifest},
		"unknown field (dsn)":    {func(m map[string]any) { m["dsn"] = "mysql://user:secret@host/db" }, snapshot.ErrManifest},
		"secret-like source id":  {func(m map[string]any) { m["source"].(map[string]any)["id"] = "mysql://u:p@db" }, snapshot.ErrManifest},
		"bad format version":     {func(m map[string]any) { m["format_version"] = 2 }, snapshot.ErrManifest},
		"non-UTC extracted_at":   {func(m map[string]any) { m["extracted_at"] = "2026-10-29T11:30:00+03:30" }, snapshot.ErrManifest},
		"bad data origin":        {func(m map[string]any) { m["data_origin"] = "production" }, snapshot.ErrManifest},
		"unsupported version":    {func(m map[string]any) { m["legacy"].(map[string]any)["version"] = "2.0.182" }, snapshot.ErrUnsupportedVersion},
		"unsupported spec":       {func(m map[string]any) { entity(m)["spec_id"] = "legacy.aggregate_customer_id.v1" }, snapshot.ErrUnsupportedSpec},
		"path traversal":         {func(m map[string]any) { entity(m)["file"] = "../../etc/passwd" }, snapshot.ErrManifest},
		"reordered columns":      {func(m map[string]any) { c := entity(m)["columns"].([]any); c[0], c[1] = c[1], c[0] }, snapshot.ErrManifest},
		"bad checksum format":    {func(m map[string]any) { entity(m)["file_sha256"] = "ABC" }, snapshot.ErrManifest},
		"bad completeness":       {func(m map[string]any) { entity(m)["completeness"] = "MOSTLY" }, snapshot.ErrManifest},
		"wrong checksum algo":    {func(m map[string]any) { m["checksum_algorithm"] = "md5" }, snapshot.ErrManifest},
		"wrong application name": {func(m map[string]any) { m["legacy"].(map[string]any)["application"] = "other" }, snapshot.ErrManifest},
	}
	for name, c := range cases {
		dir := writeOne(t, mappingID, [][]snapshot.Cell{mapping("1")}, snapshot.Complete)
		editManifest(t, dir, c.edit)
		if _, err := snapshot.Open(dir, reg()); !errors.Is(err, c.want) {
			t.Errorf("%s: err = %v, want %v", name, err, c.want)
		}
	}
	trailing := writeOne(t, mappingID, [][]snapshot.Cell{mapping("1")}, snapshot.Complete)
	f, _ := os.OpenFile(filepath.Join(trailing, snapshot.ManifestFile), os.O_APPEND|os.O_WRONLY, 0)
	_, _ = f.WriteString(`{"again":1}`)
	f.Close()
	if _, err := snapshot.Open(trailing, reg()); !errors.Is(err, snapshot.ErrManifest) {
		t.Errorf("trailing data: %v", err)
	}
}

func validateOne(t *testing.T, dir string) snapshot.EntityReport {
	t.Helper()
	p, err := snapshot.Open(dir, reg())
	if err != nil {
		t.Fatal(err)
	}
	reps, err := p.Validate()
	if err != nil {
		t.Fatal(err)
	}
	return reps[0]
}

func TestChecksumAndCountTampering(t *testing.T) {
	dir := writeOne(t, mappingID, [][]snapshot.Cell{mapping("1"), mapping("2")}, snapshot.Complete)
	file := filepath.Join(dir, snapshot.EntityFile(mappingID))
	b, _ := os.ReadFile(file)
	tampered := bytes.Replace(b, []byte("synthetic_role"), []byte("synthetic_rolf"), 1)
	if err := os.WriteFile(file, tampered, 0o600); err != nil {
		t.Fatal(err)
	}
	if r := validateOne(t, dir); r.Status != result.Failed || !strings.Contains(r.Problem, "file checksum") {
		t.Fatalf("tampered file: %+v", r)
	}
	if !strings.Contains(validateOne(t, dir).Problem, "checksum") || strings.Contains(validateOne(t, dir).Problem, "synthetic_rol") {
		t.Fatal("report must not leak row content")
	}

	counted := writeOne(t, mappingID, [][]snapshot.Cell{mapping("1"), mapping("2")}, snapshot.Complete)
	editManifest(t, counted, func(m map[string]any) { entity(m)["row_count"] = 3 })
	if r := validateOne(t, counted); r.Status != result.Failed || !strings.Contains(r.Problem, "row count") {
		t.Fatalf("count mismatch: %+v", r)
	}
}

func TestStructuralRowFailures(t *testing.T) {
	cases := map[string]func(dir, file string){
		"missing trailing newline": func(dir, file string) {
			b, _ := os.ReadFile(file)
			_ = os.WriteFile(file, bytes.TrimSuffix(b, []byte("\n")), 0o600)
		},
		"crlf line": func(dir, file string) {
			b, _ := os.ReadFile(file)
			_ = os.WriteFile(file, bytes.Replace(b, []byte("\n"), []byte("\r\n"), 1), 0o600)
		},
		"blank line": func(dir, file string) {
			b, _ := os.ReadFile(file)
			_ = os.WriteFile(file, append(b, '\n'), 0o600)
		},
	}
	for name, mutate := range cases {
		dir := writeOne(t, mappingID, [][]snapshot.Cell{mapping("1"), mapping("2")}, snapshot.Complete)
		file := filepath.Join(dir, snapshot.EntityFile(mappingID))
		mutate(dir, file)
		if r := validateOne(t, dir); r.Status != result.Failed {
			t.Errorf("%s: status %s", name, r.Status)
		}
	}
}

func TestKeyOrderingAndFormatAreEnforced(t *testing.T) {
	reg := reg()
	for name, rows := range map[string][][]snapshot.Cell{
		"descending": {mapping("2"), mapping("1")},
		"duplicate":  {mapping("1"), mapping("1")},
		"leading 0":  {mapping("01")},
		"null key":   {append([]snapshot.Cell{snapshot.Null()}, mapping("1")[1:]...)},
		"non-digit":  {mapping("1a")},
	} {
		dir := filepath.Join(t.TempDir(), "p")
		if _, err := snapshot.Write(dir, reg, legacytest.Header("synthetic-snapshot-0003", t0),
			[]snapshot.EntityRows{{SpecID: mappingID, Rows: rows}}); !errors.Is(err, snapshot.ErrWrite) {
			t.Errorf("%s: writer accepted: %v", name, err)
		}
	}
	// Numeric (not lexicographic) ordering: 9 < 10.
	if r := validateOne(t, writeOne(t, mappingID, [][]snapshot.Cell{mapping("9"), mapping("10")}, snapshot.Complete)); r.Status != result.Success {
		t.Fatalf("numeric order: %+v", r)
	}
}

func TestLayoutIsStrict(t *testing.T) {
	extra := writeOne(t, mappingID, [][]snapshot.Cell{mapping("1")}, snapshot.Complete)
	_ = os.WriteFile(filepath.Join(extra, snapshot.TablesDir, "undeclared.ndjson"), []byte("[]\n"), 0o600)
	if _, err := snapshot.Open(extra, reg()); !errors.Is(err, snapshot.ErrPackageLayout) {
		t.Errorf("undeclared file: %v", err)
	}
	missing := writeOne(t, mappingID, [][]snapshot.Cell{mapping("1")}, snapshot.Complete)
	_ = os.Remove(filepath.Join(missing, snapshot.EntityFile(mappingID)))
	if _, err := snapshot.Open(missing, reg()); !errors.Is(err, snapshot.ErrPackageLayout) {
		t.Errorf("missing file: %v", err)
	}
	root := writeOne(t, mappingID, [][]snapshot.Cell{mapping("1")}, snapshot.Complete)
	_ = os.WriteFile(filepath.Join(root, "notes.txt"), []byte("x"), 0o600)
	if _, err := snapshot.Open(root, reg()); !errors.Is(err, snapshot.ErrPackageLayout) {
		t.Errorf("root entry: %v", err)
	}
	link := writeOne(t, mappingID, [][]snapshot.Cell{mapping("1")}, snapshot.Complete)
	target := filepath.Join(link, snapshot.EntityFile(mappingID))
	moved := target + ".real"
	_ = os.Rename(target, moved)
	if err := os.Symlink(moved, target); err == nil {
		if _, err := snapshot.Open(link, reg()); !errors.Is(err, snapshot.ErrPackageLayout) {
			t.Errorf("symlinked table file: %v", err)
		}
	}
}

func TestNullInNotNullIsReportedNotRejected(t *testing.T) {
	row := mapping("1")
	row[1] = snapshot.Null() // legacy_role is NOT NULL per 2.0.160 evidence
	r := validateOne(t, writeOne(t, mappingID, [][]snapshot.Cell{row}, snapshot.Complete))
	if r.Status != result.Success || r.NullInNotNull["legacy_role"] != 1 || len(r.Warnings) != 1 {
		t.Fatalf("null-in-not-null: %+v", r)
	}
}

func TestCompletenessIsNeverSuccess(t *testing.T) {
	for c, want := range map[snapshot.Completeness]result.Status{
		snapshot.Complete: result.Success, snapshot.Incomplete: result.Partial,
		snapshot.CompletenessUnknown: result.Unknown, snapshot.NotRecoverable: result.Partial,
	} {
		r := validateOne(t, writeOne(t, mappingID, [][]snapshot.Cell{mapping("1")}, c))
		if r.Status != want {
			t.Errorf("%s: %s, want %s", c, r.Status, want)
		}
	}
}

func TestEmptyEntity(t *testing.T) {
	r := validateOne(t, writeOne(t, mappingID, nil, snapshot.Complete))
	if r.Status != result.Success || r.ParsedRows != 0 || r.ActualBytes != 0 {
		t.Fatalf("empty: %+v", r)
	}
}
