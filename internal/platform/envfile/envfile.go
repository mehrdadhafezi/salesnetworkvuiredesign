// Package envfile resolves configuration that may be injected as a file
// (R3 secret injection): for a key K, a non-empty K wins; otherwise, when
// K_FILE names a file, its content (trailing newline trimmed) is used.
// Secret managers and container runtimes mount secrets as files
// (/run/secrets/...), so credentials never need to live in environment
// variables, images or command lines.
package envfile

import (
	"os"
	"strings"
)

// MaxSize bounds a secret file.
const MaxSize = 64 << 10

// Getenv wraps get with the K_FILE fallback. An unreadable, oversized or
// world-readable secret file yields "" (the caller's validation then fails
// closed); its content is never logged.
func Getenv(get func(string) string) func(string) string {
	return func(k string) string {
		if v := get(k); v != "" {
			return v
		}
		p := get(k + "_FILE")
		if p == "" {
			return ""
		}
		fi, err := os.Stat(p)
		if err != nil || !fi.Mode().IsRegular() || fi.Size() > MaxSize || fi.Mode().Perm()&0o007 != 0 {
			return ""
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return ""
		}
		return strings.TrimRight(string(b), "\r\n")
	}
}
