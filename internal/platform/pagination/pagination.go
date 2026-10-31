// Package pagination implements opaque keyset cursors for read endpoints
// (Architecture V1 §23: cursor-based pagination, no totals). A cursor binds
// the sort it was issued for: replaying it with another sort is rejected, so
// a client can never splice two orderings. Cursors carry only the last row's
// sort value and native id, never business data beyond what the page showed.
package pagination

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
)

// ErrInvalidCursor reports a malformed, oversized or foreign cursor.
var ErrInvalidCursor = errors.New("pagination: invalid cursor")

// MaxCursorLen bounds an encoded cursor (contract maxLength).
const MaxCursorLen = 512

// Default and maximum page sizes (contract Limit parameter).
const (
	DefaultLimit = 50
	MaxLimit     = 200
)

// Cursor is the decoded position after the last returned row.
type Cursor struct {
	Sort  string `json:"s"`
	Value string `json:"v"`
	ID    string `json:"i"`
}

// Encode returns the opaque form of c.
func Encode(c Cursor) (string, error) {
	b, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	s := base64.RawURLEncoding.EncodeToString(b)
	if len(s) > MaxCursorLen {
		return "", fmt.Errorf("%w: too long", ErrInvalidCursor)
	}
	return s, nil
}

// Decode parses s and checks it was issued for sort.
func Decode(s, sort string) (Cursor, error) {
	if s == "" || len(s) > MaxCursorLen {
		return Cursor{}, ErrInvalidCursor
	}
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return Cursor{}, ErrInvalidCursor
	}
	var c Cursor
	if err := json.Unmarshal(b, &c); err != nil || c.ID == "" {
		return Cursor{}, ErrInvalidCursor
	}
	if c.Sort != sort {
		return Cursor{}, fmt.Errorf("%w: issued for another sort", ErrInvalidCursor)
	}
	return c, nil
}

// Limit normalises a requested page size; zero or nil means DefaultLimit.
func Limit(n *int) (int, error) {
	if n == nil {
		return DefaultLimit, nil
	}
	if *n < 1 || *n > MaxLimit {
		return 0, fmt.Errorf("pagination: limit %d out of range", *n)
	}
	return *n, nil
}
