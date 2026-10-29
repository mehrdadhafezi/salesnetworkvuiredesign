// Package snapshot defines the deterministic legacy snapshot package and
// validates it offline (Architecture V1 §24.2–3). It never connects to a
// legacy system: a package is a directory supplied intentionally by an
// operator (or built from synthetic fixtures in tests).
//
// # Package layout
//
//	<dir>/manifest.json
//	<dir>/tables/<spec-id>.ndjson      one file per declared entity
//
// # Row encoding (format version 1)
//
// Each table file is UTF-8 NDJSON: one row per line, every line terminated by
// a single "\n" (no CR, no BOM, no blank lines). A row is a JSON array with
// exactly one cell per spec column, in spec order. A cell is either JSON null
// (SQL NULL) or a JSON string holding the raw legacy value as text. Numbers,
// booleans, objects and nested arrays are rejected, so NULL, the empty string
// and the text "null" stay distinct, and no value is ever re-typed.
//
// Lines must be in canonical form, the exact bytes produced by EncodeRow:
// only `"` `\` and control characters below U+0020 are escaped (\b \f \n \r \t
// in short form, others as \u00XX in lowercase hex); every other character,
// including non-ASCII text, is emitted as raw UTF-8; there is no whitespace.
// A non-canonical line is rejected rather than rewritten, so the raw file
// checksum and the content checksum describe the same bytes. Strings must be
// valid UTF-8 and must not contain U+0000 (PostgreSQL text cannot store it);
// such rows fail validation explicitly.
package snapshot

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"strconv"
	"unicode/utf8"
)

// Cell is one raw value: Null, or Text (which may be the empty string).
type Cell struct {
	Null bool
	Text string
}

// Text returns a non-NULL cell.
func Text(s string) Cell { return Cell{Text: s} }

// Null returns a NULL cell.
func Null() Cell { return Cell{Null: true} }

// Row-level decoding failures. Each is reported with the entity and ordinal,
// never with the row content.
var (
	ErrNotArray         = errors.New("snapshot: row is not a JSON array")
	ErrCellType         = errors.New("snapshot: cell is neither null nor a string")
	ErrColumnCount      = errors.New("snapshot: row has the wrong number of cells")
	ErrInvalidUTF8      = errors.New("snapshot: cell is not valid UTF-8")
	ErrNULCharacter     = errors.New("snapshot: cell contains U+0000, which staging cannot store")
	ErrNonCanonicalLine = errors.New("snapshot: line is not in canonical encoding")
)

const hexDigits = "0123456789abcdef"

// EncodeRow returns the canonical encoding of one row (without the newline).
// It fails on invalid UTF-8 or U+0000 instead of altering the value.
func EncodeRow(cells []Cell) ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte('[')
	for i, c := range cells {
		if i > 0 {
			b.WriteByte(',')
		}
		if c.Null {
			b.WriteString("null")
			continue
		}
		if !utf8.ValidString(c.Text) {
			return nil, fmt.Errorf("%w (cell %d)", ErrInvalidUTF8, i)
		}
		b.WriteByte('"')
		for j := 0; j < len(c.Text); j++ {
			ch := c.Text[j]
			switch {
			case ch == 0:
				return nil, fmt.Errorf("%w (cell %d)", ErrNULCharacter, i)
			case ch == '"':
				b.WriteString(`\"`)
			case ch == '\\':
				b.WriteString(`\\`)
			case ch == '\b':
				b.WriteString(`\b`)
			case ch == '\f':
				b.WriteString(`\f`)
			case ch == '\n':
				b.WriteString(`\n`)
			case ch == '\r':
				b.WriteString(`\r`)
			case ch == '\t':
				b.WriteString(`\t`)
			case ch < 0x20:
				b.WriteString(`\u00`)
				b.WriteByte(hexDigits[ch>>4])
				b.WriteByte(hexDigits[ch&0x0f])
			default:
				b.WriteByte(ch)
			}
		}
		b.WriteByte('"')
	}
	b.WriteByte(']')
	return b.Bytes(), nil
}

// DecodeRow parses one line (without its newline) into exactly width cells
// and proves the line is canonical by re-encoding it.
func DecodeRow(line []byte, width int) ([]Cell, error) {
	var raw []json.RawMessage
	if err := json.Unmarshal(line, &raw); err != nil || raw == nil {
		return nil, ErrNotArray
	}
	if len(raw) != width {
		return nil, fmt.Errorf("%w: got %d, want %d", ErrColumnCount, len(raw), width)
	}
	cells := make([]Cell, width)
	for i, r := range raw {
		if bytes.Equal(r, []byte("null")) {
			cells[i] = Null()
			continue
		}
		if len(r) == 0 || r[0] != '"' {
			return nil, fmt.Errorf("%w (cell %d)", ErrCellType, i)
		}
		var s string
		if err := json.Unmarshal(r, &s); err != nil {
			return nil, fmt.Errorf("%w (cell %d)", ErrCellType, i)
		}
		cells[i] = Text(s)
	}
	canon, err := EncodeRow(cells)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(canon, line) {
		// Covers escaped forms of characters that must be raw, invalid UTF-8
		// that the JSON decoder replaced, whitespace and key-order drift.
		if !utf8.Valid(line) {
			return nil, ErrInvalidUTF8
		}
		return nil, ErrNonCanonicalLine
	}
	return cells, nil
}

// ContentDomain prefixes every content checksum (domain separation).
const ContentDomain = "crm-legacy-content-v1"

// ContentHasher computes the canonical content checksum of one entity:
//
//	SHA-256( "crm-legacy-content-v1\n" <spec-id> "\n"
//	         then, for each row in source-ordinal order:
//	         <decimal byte length of the canonical row> ":" <canonical row> "\n" )
//
// It depends only on the spec ID and the ordered canonical rows: no
// timestamps, file names or batch identity enter the hash, so identical
// content always yields the identical checksum, and staging can recompute it
// from the stored raw values.
type ContentHasher struct {
	h    hash.Hash
	rows int64
}

// NewContentHasher starts a content checksum for specID.
func NewContentHasher(specID string) *ContentHasher {
	h := sha256.New()
	h.Write([]byte(ContentDomain + "\n" + specID + "\n"))
	return &ContentHasher{h: h}
}

// AddRow adds one canonical row encoding.
func (c *ContentHasher) AddRow(canonical []byte) {
	c.h.Write([]byte(strconv.Itoa(len(canonical))))
	c.h.Write([]byte{':'})
	c.h.Write(canonical)
	c.h.Write([]byte{'\n'})
	c.rows++
}

// Rows returns the number of rows added.
func (c *ContentHasher) Rows() int64 { return c.rows }

// Sum returns the lowercase hex checksum.
func (c *ContentHasher) Sum() string { return hex.EncodeToString(c.h.Sum(nil)) }

// RowSHA256 returns the SHA-256 of one canonical row (per-row provenance).
func RowSHA256(canonical []byte) []byte {
	s := sha256.Sum256(canonical)
	return s[:]
}
