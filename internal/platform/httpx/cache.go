package httpx

import (
	"net/http"
	"strings"
)

// RM-E04B / A2 (F-07, NT-17): one HTTP cache-prevention boundary.
//
// Policy, applied by CachePolicy to every response it wraps:
//
//   - a request path under /api (APIPath) is private: the response always
//     carries exactly `Cache-Control: no-store`, `Pragma: no-cache` and a
//     Vary that includes Cookie (every /api operation is evaluated against
//     the caller's session cookie, so the same URL answers differently per
//     caller — data, 401 or 403);
//   - any other response is left alone unless it sets a cookie: a response
//     carrying Set-Cookie is private too, wherever it is served from;
//   - static assets and other non-API responses keep their caching.
//
// Header precedence: the policy is applied before the inner handler runs
// and normalised again when the final status is written, so a handler (or
// an inner middleware, or a panic recovery) that sets a conflicting or
// duplicate Cache-Control cannot win. Cache-Control and Pragma are
// replaced, never appended. Vary is merged: existing tokens are kept in
// order, duplicates (case-insensitive) are removed and Cookie appears once,
// last;
// `Vary: *` is kept as the single value it must be. Applying the
// middleware more than once gives the same headers as applying it once.
//
// The policy concerns HTTP caching only. It does not control browser
// history / back-forward cache behaviour or intermediaries that ignore
// Cache-Control, and it cannot reach responses net/http writes before any
// handler runs (malformed requests answered by the server itself).

// APIPath reports whether path is under the /api prefix ("/api" or
// "/api/..."; "/apix" is not).
func APIPath(path string) bool {
	return path == "/api" || strings.HasPrefix(path, "/api/")
}

// MarkPrivate sets the no-store policy on h: Cache-Control and Pragma are
// replaced and Cookie is merged into Vary. It is idempotent.
func MarkPrivate(h http.Header) {
	h.Set("Cache-Control", "no-store")
	h.Set("Pragma", "no-cache") // HTTP/1.0 intermediaries
	h["Vary"] = mergeVary(h.Values("Vary"), "Cookie")
}

// mergeVary returns the Vary field as one value: existing tokens in order,
// without case-insensitive duplicates, then add (last, once; an existing
// spelling of it is kept). "*" absorbs every other token. Keeping add last
// makes the result independent of whether it was merged before or after
// the handler added its own tokens.
func mergeVary(values []string, add string) []string {
	var out []string
	last := add
	seen := map[string]bool{}
	for _, v := range values {
		for _, tok := range strings.Split(v, ",") {
			tok = strings.TrimSpace(tok)
			switch k := strings.ToLower(tok); {
			case tok == "":
			case tok == "*":
				return []string{"*"}
			case k == strings.ToLower(add):
				last = tok
			case !seen[k]:
				seen[k] = true
				out = append(out, tok)
			}
		}
	}
	return []string{strings.Join(append(out, last), ", ")}
}

// CachePolicy is the cache-prevention middleware described above. Install it
// outside every middleware that can answer early (correlation, recovery,
// body limits, session / CSRF guards, routing) so those responses are
// covered as well.
func CachePolicy(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api := APIPath(r.URL.Path)
		if api {
			MarkPrivate(w.Header())
		}
		next.ServeHTTP(&cacheWriter{ResponseWriter: w, api: api}, r)
	})
}

// cacheWriter re-applies the policy when the final status is written.
type cacheWriter struct {
	http.ResponseWriter
	api  bool
	done bool
}

func (c *cacheWriter) finalise() {
	if c.done {
		return
	}
	c.done = true
	h := c.ResponseWriter.Header()
	if c.api || len(h.Values("Set-Cookie")) > 0 {
		MarkPrivate(h)
	}
}

func (c *cacheWriter) WriteHeader(code int) {
	// 1xx informational responses are not the final response; headers may
	// still change after them.
	if code >= 200 {
		c.finalise()
	}
	c.ResponseWriter.WriteHeader(code)
}

func (c *cacheWriter) Write(b []byte) (int, error) {
	c.finalise()
	return c.ResponseWriter.Write(b)
}

// Flush keeps streaming support of the underlying writer, if any.
func (c *cacheWriter) Flush() {
	c.finalise()
	if f, ok := c.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (c *cacheWriter) Unwrap() http.ResponseWriter { return c.ResponseWriter }
