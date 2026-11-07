// Package password verifies stored password credentials (R1). It uses the
// established Argon2id KDF from golang.org/x/crypto (RFC 9106) with
// PHC-string encoding; it implements no cryptographic primitive itself.
//
// Policy boundaries (decisions open, nothing invented here):
//   - there is no password strength, length minimum, rotation, history,
//     expiry, recovery or reset rule: credential enrolment and the initial
//     password policy are D-34 (BLOCKED). Hash exists for that future
//     approved enrolment and for tests; no production path calls it;
//   - legacy WordPress hashes are never accepted (only $argon2id$ verifies).
//
// Technical controls (security sign-off requested, not policy):
//   - default parameters are the OWASP Password Storage Cheat Sheet baseline
//     for Argon2id (m=19 MiB, t=2, p=1);
//   - stored parameters are bounded so a tampered row cannot make a single
//     verification exhaust memory or CPU;
//   - candidate passwords longer than MaxPasswordBytes are rejected before
//     hashing (denial-of-service bound, not a password policy);
//   - comparison is constant-time.
package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Params are Argon2id cost parameters.
type Params struct {
	MemoryKiB   uint32
	Iterations  uint32
	Parallelism uint8
	SaltBytes   int
	KeyBytes    uint32
}

// Default is the OWASP Argon2id baseline.
var Default = Params{MemoryKiB: 19 * 1024, Iterations: 2, Parallelism: 1, SaltBytes: 16, KeyBytes: 32}

// Bounds accepted when parsing a stored hash.
const (
	maxMemoryKiB   = 256 * 1024
	maxIterations  = 10
	maxParallelism = 8
	maxKeyBytes    = 64
	// MaxPasswordBytes bounds the work done for one candidate.
	MaxPasswordBytes = 1024
)

// ErrMalformed reports a stored hash that is not a bounded $argon2id$ PHC string.
var ErrMalformed = errors.New("password: stored hash is not a supported argon2id PHC string")

// Hash derives a PHC-encoded Argon2id hash. Only an approved enrolment
// command (none exists; D-34) and tests may call it.
func Hash(candidate string, p Params) (string, error) {
	if len(candidate) == 0 || len(candidate) > MaxPasswordBytes {
		return "", errors.New("password: candidate length out of bounds")
	}
	salt := make([]byte, p.SaltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(candidate), salt, p.Iterations, p.MemoryKiB, p.Parallelism, p.KeyBytes)
	enc := base64.RawStdEncoding
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, p.MemoryKiB, p.Iterations, p.Parallelism,
		enc.EncodeToString(salt), enc.EncodeToString(key)), nil
}

type parsed struct {
	params Params
	salt   []byte
	key    []byte
}

func parse(phc string) (parsed, error) {
	parts := strings.Split(phc, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" || parts[2] != "v="+strconv.Itoa(argon2.Version) {
		return parsed{}, ErrMalformed
	}
	var m, t, p uint64
	for _, kv := range strings.Split(parts[3], ",") {
		k, v, ok := strings.Cut(kv, "=")
		n, err := strconv.ParseUint(v, 10, 32)
		if !ok || err != nil {
			return parsed{}, ErrMalformed
		}
		switch k {
		case "m":
			m = n
		case "t":
			t = n
		case "p":
			p = n
		default:
			return parsed{}, ErrMalformed
		}
	}
	enc := base64.RawStdEncoding
	salt, err1 := enc.DecodeString(parts[4])
	key, err2 := enc.DecodeString(parts[5])
	if err1 != nil || err2 != nil || len(salt) < 8 || len(key) < 16 || len(key) > maxKeyBytes ||
		m < 8 || m > maxMemoryKiB || t < 1 || t > maxIterations || p < 1 || p > maxParallelism {
		return parsed{}, ErrMalformed
	}
	return parsed{params: Params{MemoryKiB: uint32(m), Iterations: uint32(t), Parallelism: uint8(p), SaltBytes: len(salt), KeyBytes: uint32(len(key))},
		salt: salt, key: key}, nil
}

// Verify reports whether candidate matches the stored PHC hash. A malformed
// hash or an out-of-bounds candidate never matches.
func Verify(phc, candidate string) (bool, error) {
	if len(candidate) == 0 || len(candidate) > MaxPasswordBytes {
		// Still spend comparable work so length is not a timing oracle.
		Dummy(candidate)
		return false, nil
	}
	pp, err := parse(phc)
	if err != nil {
		Dummy(candidate)
		return false, err
	}
	got := argon2.IDKey([]byte(candidate), pp.salt, pp.params.Iterations, pp.params.MemoryKiB, pp.params.Parallelism, pp.params.KeyBytes)
	return subtle.ConstantTimeCompare(got, pp.key) == 1, nil
}

var dummySalt = []byte("crm-r1-dummy-salt")

// Dummy spends one default-cost derivation and discards it, so an unknown
// login or a disabled credential costs the same time as a real check
// (no account enumeration by timing).
func Dummy(candidate string) {
	if len(candidate) > MaxPasswordBytes {
		candidate = candidate[:MaxPasswordBytes]
	}
	_ = argon2.IDKey([]byte(candidate), dummySalt, Default.Iterations, Default.MemoryKiB, Default.Parallelism, Default.KeyBytes)
}
