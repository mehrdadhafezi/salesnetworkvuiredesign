// Package ids generates technical identifiers for platform records
// (outbox events, audit events). They are random UUIDs, never derived from
// business data such as phone numbers.
package ids

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"regexp"
)

// ErrInvalidUUID reports a malformed UUID string.
var ErrInvalidUUID = errors.New("ids: invalid uuid")

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// NewUUID returns a random RFC 9562 version 4 UUID in canonical lowercase form.
func NewUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("ids: random source: %w", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

// ValidateUUID checks the canonical lowercase UUID form.
func ValidateUUID(s string) error {
	if !uuidPattern.MatchString(s) {
		return fmt.Errorf("%w: %q", ErrInvalidUUID, s)
	}
	return nil
}

// Stable returns a deterministic UUID (version-5 layout over SHA-256) of a
// technical namespace and its parts, for command identities that must be the
// same on every retry of the same intent (e.g. scope + idempotency key). Callers
// never pass phone numbers, names or timestamps as the only identity.
func Stable(namespace string, parts ...string) string {
	h := sha256.New()
	h.Write([]byte(namespace))
	for _, p := range parts {
		h.Write([]byte{0})
		h.Write([]byte(p))
	}
	b := h.Sum(nil)[:16]
	b[6] = (b[6] & 0x0f) | 0x50
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
