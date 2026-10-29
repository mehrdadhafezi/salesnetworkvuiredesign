package snapshot

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/spec"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
)

// MaxLineBytes bounds one encoded row.
const MaxLineBytes = 64 << 20

// Structural package failures.
var (
	ErrPackageLayout  = errors.New("snapshot: invalid package layout")
	ErrMissingNewline = errors.New("snapshot: last line has no trailing newline")
	ErrLineTooLong    = errors.New("snapshot: line exceeds the maximum row size")
	ErrKeyFormat      = errors.New("snapshot: key is not a canonical unsigned decimal")
	ErrKeyOrder       = errors.New("snapshot: keys are not strictly ascending (duplicate or out of order)")
)

// RowError locates a failed row without exposing its content.
type RowError struct {
	SpecID  string
	Ordinal int64
	Err     error
}

func (e *RowError) Error() string {
	return fmt.Sprintf("%s ordinal %d: %v", e.SpecID, e.Ordinal, e.Err)
}

func (e *RowError) Unwrap() error { return e.Err }

// Package is an opened snapshot directory with a parsed manifest.
type Package struct {
	Dir            string
	Manifest       Manifest
	ManifestSHA256 string
	reg            *spec.Registry
}

// Open reads and validates the manifest of dir and the package layout:
// exactly manifest.json and tables/<declared files>, regular files only.
// Entity table contents are checked by ValidateEntity.
func Open(dir string, reg *spec.Registry) (*Package, error) {
	b, err := readRegular(filepath.Join(dir, ManifestFile), maxManifestBytes)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPackageLayout, err)
	}
	m, err := ParseManifest(b)
	if err != nil {
		return nil, err
	}
	if err := m.Validate(reg); err != nil {
		return nil, err
	}
	if err := checkLayout(dir, m); err != nil {
		return nil, err
	}
	sum := sha256.Sum256(b)
	return &Package{Dir: dir, Manifest: m, ManifestSHA256: hex.EncodeToString(sum[:]), reg: reg}, nil
}

// Spec returns the spec of a declared entity.
func (p *Package) Spec(specID string) (spec.Spec, error) {
	return p.reg.ForVersion(specID, p.Manifest.Legacy.Version)
}

func readRegular(path string, limit int64) ([]byte, error) {
	fi, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a regular file", filepath.Base(path))
	}
	if fi.Size() > limit {
		return nil, fmt.Errorf("%s exceeds %d bytes", filepath.Base(path), limit)
	}
	return os.ReadFile(path)
}

func checkLayout(dir string, m Manifest) error {
	top, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPackageLayout, err)
	}
	for _, e := range top {
		if e.Name() == ManifestFile && e.Type().IsRegular() {
			continue
		}
		if e.Name() == TablesDir && e.IsDir() {
			continue
		}
		return fmt.Errorf("%w: unexpected entry %q at package root", ErrPackageLayout, e.Name())
	}
	declared := map[string]bool{}
	for _, e := range m.Entities {
		declared[filepath.Base(e.File)] = true
	}
	entries, err := os.ReadDir(filepath.Join(dir, TablesDir))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPackageLayout, err)
	}
	found := map[string]bool{}
	for _, e := range entries {
		if !e.Type().IsRegular() || !declared[e.Name()] {
			return fmt.Errorf("%w: undeclared or non-regular entry tables/%s", ErrPackageLayout, e.Name())
		}
		found[e.Name()] = true
	}
	for name := range declared {
		if !found[name] {
			return fmt.Errorf("%w: declared file tables/%s is missing", ErrPackageLayout, name)
		}
	}
	return nil
}

// Row is one decoded row handed to a visitor.
type Row struct {
	Ordinal   int64 // 1-based position in the file
	Cells     []Cell
	Canonical []byte // canonical encoding (equal to the line bytes)
	Key       string // raw source key text
}

// EntityReport is the technical validation/reconciliation result for one
// entity. It never contains row values.
type EntityReport struct {
	SpecID             string            `json:"spec_id"`
	SourceTable        string            `json:"source_table"`
	Classification     string            `json:"classification"`
	Completeness       Completeness      `json:"completeness"`
	ExpectedRows       int64             `json:"expected_rows"`
	ParsedRows         int64             `json:"parsed_rows"`
	ExpectedBytes      int64             `json:"expected_bytes"`
	ActualBytes        int64             `json:"actual_bytes"`
	ExpectedFileSHA256 string            `json:"expected_file_sha256"`
	ActualFileSHA256   string            `json:"actual_file_sha256,omitempty"`
	ExpectedContentSHA string            `json:"expected_content_sha256"`
	ActualContentSHA   string            `json:"actual_content_sha256,omitempty"`
	NullInNotNull      map[string]int64  `json:"null_in_not_null,omitempty"`
	Status             result.Status     `json:"status"`
	Reason             *reasoncodes.Code `json:"reason_code,omitempty"`
	Problem            string            `json:"problem,omitempty"`
	Warnings           []string          `json:"warnings"`
}

var unsignedDecimal = regexp.MustCompile(`^(0|[1-9][0-9]*)$`)

// keyLess compares two canonical unsigned decimals numerically.
func keyLess(a, b string) bool {
	if len(a) != len(b) {
		return len(a) < len(b)
	}
	return a < b
}

// ScanEntity streams the entity file once, checking framing, canonical
// encoding, width, UTF-8, U+0000, key format and strict key order, and calls
// visit for every row. It returns the report; the report Status is SUCCESS
// only if every declared count and checksum matches what was read. visit
// errors abort the scan and are returned.
func (p *Package) ScanEntity(e Entity, visit func(Row) error) (EntityReport, error) {
	s, err := p.Spec(e.SpecID)
	if err != nil {
		return EntityReport{}, err
	}
	rep := EntityReport{
		SpecID: e.SpecID, SourceTable: s.SourceTable, Classification: string(s.Classification()),
		Completeness: e.Completeness, ExpectedRows: e.RowCount, ExpectedBytes: e.ByteSize,
		ExpectedFileSHA256: e.FileSHA256, ExpectedContentSHA: e.ContentSHA256, Warnings: []string{},
	}
	fail := func(code reasoncodes.Code, problem string) (EntityReport, error) {
		rep.Status, rep.Reason, rep.Problem = result.Failed, &code, problem
		return rep, nil
	}
	f, err := os.Open(filepath.Join(p.Dir, filepath.FromSlash(e.File)))
	if err != nil {
		return fail(reasoncodes.SourceUnavailable, "table file cannot be opened")
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil || !fi.Mode().IsRegular() {
		return fail(reasoncodes.SourceUnavailable, "table file is not a regular file")
	}

	fileHash := sha256.New()
	content := NewContentHasher(e.SpecID)
	keyIdx := s.KeyIndexes()[0]
	notNull := map[int]string{}
	for i, c := range s.Columns {
		if c.Nullability == spec.NotNull {
			notNull[i] = c.Name
		}
	}
	nulls := map[string]int64{}
	var prevKey string
	var ordinal, bytesRead int64

	r := bufio.NewReaderSize(io.TeeReader(f, fileHash), 1<<20)
	for {
		line, err := readLine(r)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fail(reasoncodes.ContractInvalid, (&RowError{SpecID: e.SpecID, Ordinal: ordinal + 1, Err: err}).Error())
		}
		ordinal++
		bytesRead += int64(len(line)) + 1
		cells, err := DecodeRow(line, len(s.Columns))
		if err != nil {
			return fail(reasoncodes.ContractInvalid, (&RowError{SpecID: e.SpecID, Ordinal: ordinal, Err: err}).Error())
		}
		key := cells[keyIdx]
		if key.Null || !unsignedDecimal.MatchString(key.Text) {
			return fail(reasoncodes.ContractInvalid, (&RowError{SpecID: e.SpecID, Ordinal: ordinal, Err: ErrKeyFormat}).Error())
		}
		if ordinal > 1 && !keyLess(prevKey, key.Text) {
			return fail(reasoncodes.ContractInvalid, (&RowError{SpecID: e.SpecID, Ordinal: ordinal, Err: ErrKeyOrder}).Error())
		}
		prevKey = key.Text
		for i, name := range notNull {
			if cells[i].Null {
				nulls[name]++
			}
		}
		content.AddRow(line)
		if visit != nil {
			if err := visit(Row{Ordinal: ordinal, Cells: cells, Canonical: line, Key: key.Text}); err != nil {
				return rep, err
			}
		}
	}
	rep.ParsedRows, rep.ActualBytes = ordinal, bytesRead
	rep.ActualFileSHA256 = hex.EncodeToString(fileHash.Sum(nil))
	rep.ActualContentSHA = content.Sum()
	if len(nulls) > 0 {
		rep.NullInNotNull = nulls
		names := make([]string, 0, len(nulls))
		for n := range nulls {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			rep.Warnings = append(rep.Warnings, fmt.Sprintf("column %s: %d NULL value(s) where the 2.0.160 evidence expects NOT NULL; preserved raw", n, nulls[n]))
		}
	}
	switch {
	case rep.ActualFileSHA256 != e.FileSHA256:
		return fail(reasoncodes.ContractInvalid, "file checksum does not match the manifest")
	case rep.ActualBytes != e.ByteSize:
		return fail(reasoncodes.ContractInvalid, "byte size does not match the manifest")
	case rep.ParsedRows != e.RowCount:
		return fail(reasoncodes.ContractInvalid, "row count does not match the manifest")
	case rep.ActualContentSHA != e.ContentSHA256:
		return fail(reasoncodes.ContractInvalid, "content checksum does not match the manifest")
	}
	rep.Status = CompletenessStatus(e.Completeness)
	if rep.Status != result.Success {
		code := reasoncodes.CoverageIncomplete
		rep.Reason = &code
		rep.Problem = "manifest declares completeness " + string(e.Completeness)
	}
	return rep, nil
}

// CompletenessStatus maps a verified entity's declared completeness to its
// outcome: COMPLETE → SUCCESS, UNKNOWN → UNKNOWN, INCOMPLETE and
// NOT_RECOVERABLE → PARTIAL. An incomplete entity is never SUCCESS.
func CompletenessStatus(c Completeness) result.Status {
	switch c {
	case Complete:
		return result.Success
	case CompletenessUnknown:
		return result.Unknown
	default:
		return result.Partial
	}
}

// readLine returns one line without its "\n"; a final unterminated line is
// an error, as is a line longer than MaxLineBytes.
func readLine(r *bufio.Reader) ([]byte, error) {
	var buf []byte
	for {
		chunk, err := r.ReadSlice('\n')
		buf = append(buf, chunk...)
		if len(buf) > MaxLineBytes+1 {
			return nil, ErrLineTooLong
		}
		switch {
		case err == nil:
			return bytes.TrimSuffix(buf, []byte{'\n'}), nil
		case errors.Is(err, bufio.ErrBufferFull):
			continue
		case err == io.EOF:
			if len(buf) == 0 {
				return nil, io.EOF
			}
			return nil, ErrMissingNewline
		default:
			return nil, err
		}
	}
}

// Validate scans every entity of the package without a visitor.
func (p *Package) Validate() ([]EntityReport, error) {
	out := make([]EntityReport, 0, len(p.Manifest.Entities))
	for _, e := range p.Manifest.Entities {
		rep, err := p.ScanEntity(e, nil)
		if err != nil {
			return out, err
		}
		out = append(out, rep)
	}
	return out, nil
}
