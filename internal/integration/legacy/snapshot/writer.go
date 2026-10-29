package snapshot

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/spec"
)

// ErrWrite reports a package that cannot be written faithfully.
var ErrWrite = errors.New("snapshot: cannot write package")

// Header is the caller-supplied part of a manifest; the writer computes
// entities, counts, sizes and checksums itself.
type Header struct {
	SnapshotID  string
	Legacy      Legacy
	Source      Source
	DataOrigin  DataOrigin
	ExtractedAt string
	Tool        Tool
	Warnings    []string
}

// EntityRows is the ordered raw rows of one entity.
type EntityRows struct {
	SpecID       string
	Rows         [][]Cell
	Completeness Completeness
	Notes        []string
}

// Write creates a new package in dir (which must not exist). Rows are written
// exactly as given; out-of-order or duplicate keys are rejected instead of
// being sorted or deduplicated. The result passes Open and Validate.
func Write(dir string, reg *spec.Registry, h Header, entities []EntityRows) (Manifest, error) {
	if _, err := os.Lstat(dir); err == nil {
		return Manifest{}, fmt.Errorf("%w: %s already exists", ErrWrite, dir)
	}
	if err := os.MkdirAll(filepath.Join(dir, TablesDir), 0o700); err != nil {
		return Manifest{}, fmt.Errorf("%w: %v", ErrWrite, err)
	}
	ents := append([]EntityRows(nil), entities...)
	sort.Slice(ents, func(i, j int) bool { return ents[i].SpecID < ents[j].SpecID })
	warnings := h.Warnings
	if warnings == nil {
		warnings = []string{}
	}
	m := Manifest{
		Format: ManifestFormat, FormatVersion: ManifestFormatVersion, SnapshotID: h.SnapshotID,
		Legacy: h.Legacy, Source: h.Source, DataOrigin: h.DataOrigin, ExtractedAt: h.ExtractedAt,
		Tool: h.Tool, ChecksumAlgorithm: ChecksumAlgorithm, Warnings: warnings,
	}
	for _, er := range ents {
		s, err := reg.ForVersion(er.SpecID, h.Legacy.Version)
		if err != nil {
			return Manifest{}, fmt.Errorf("%w: %v", ErrWrite, err)
		}
		var body bytes.Buffer
		content := NewContentHasher(s.ID)
		keyIdx := s.KeyIndexes()[0]
		prev := ""
		for i, cells := range er.Rows {
			if len(cells) != len(s.Columns) {
				return Manifest{}, fmt.Errorf("%w: %s row %d: %v", ErrWrite, s.ID, i+1, ErrColumnCount)
			}
			k := cells[keyIdx]
			if k.Null || !unsignedDecimal.MatchString(k.Text) {
				return Manifest{}, fmt.Errorf("%w: %s row %d: %v", ErrWrite, s.ID, i+1, ErrKeyFormat)
			}
			if i > 0 && !keyLess(prev, k.Text) {
				return Manifest{}, fmt.Errorf("%w: %s row %d: %v", ErrWrite, s.ID, i+1, ErrKeyOrder)
			}
			prev = k.Text
			line, err := EncodeRow(cells)
			if err != nil {
				return Manifest{}, fmt.Errorf("%w: %s row %d: %v", ErrWrite, s.ID, i+1, err)
			}
			body.Write(line)
			body.WriteByte('\n')
			content.AddRow(line)
		}
		completeness := er.Completeness
		if completeness == "" {
			completeness = Complete
		}
		if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(EntityFile(s.ID))), body.Bytes(), 0o600); err != nil {
			return Manifest{}, fmt.Errorf("%w: %v", ErrWrite, err)
		}
		sum := sha256.Sum256(body.Bytes())
		m.Entities = append(m.Entities, Entity{
			SpecID: s.ID, File: EntityFile(s.ID), Columns: s.ColumnNames(),
			RowCount: int64(len(er.Rows)), ByteSize: int64(body.Len()),
			FileSHA256: hex.EncodeToString(sum[:]), ContentSHA256: content.Sum(),
			Completeness: completeness, Notes: er.Notes,
		})
	}
	if err := m.Validate(reg); err != nil {
		return Manifest{}, fmt.Errorf("%w: %v", ErrWrite, err)
	}
	b, err := MarshalManifest(m)
	if err != nil {
		return Manifest{}, err
	}
	if err := os.WriteFile(filepath.Join(dir, ManifestFile), b, 0o600); err != nil {
		return Manifest{}, fmt.Errorf("%w: %v", ErrWrite, err)
	}
	return m, nil
}

// MarshalManifest returns the deterministic manifest encoding (two-space
// indent, field order as declared, trailing newline, no HTML escaping).
func MarshalManifest(m Manifest) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(m); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrWrite, err)
	}
	return buf.Bytes(), nil
}
