package websession

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestTokenDigestAndCSRF(t *testing.T) {
	a, err := NewToken()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := NewToken()
	if len(a) != 43 || a == b || strings.ContainsAny(a, "+/=") {
		t.Fatalf("tokens: %q %q", a, b)
	}
	da, ok := Digest(a)
	if !ok || da == [32]byte{} {
		t.Fatal("digest of a valid token")
	}
	if db, _ := Digest(b); db == da {
		t.Fatal("digest collision")
	}
	for _, bad := range []string{"", "short", a + "x", strings.Repeat("!", 43), strings.Repeat("A", 44)} {
		if _, ok := Digest(bad); ok {
			t.Fatalf("%q accepted", bad)
		}
	}
	ca, cb := CSRFToken(a), CSRFToken(b)
	if ca == cb || ca == a || len(ca) != 43 || CSRFToken(a) != ca {
		t.Fatal("csrf token must be deterministic per session and differ across sessions")
	}
	if !CSRFValid(ca, ca) || CSRFValid(ca, cb) || CSRFValid(ca, "") || CSRFValid("", "") {
		t.Fatal("csrf comparison")
	}
}

func TestCookieAttributes(t *testing.T) {
	now := time.Date(2026, 11, 7, 9, 0, 0, 0, time.UTC)
	c := Cookie("tok", now.Add(2*time.Hour), now)
	if c.Name != "__Host-crm_session" || !c.Secure || !c.HttpOnly || c.SameSite != http.SameSiteStrictMode || c.Path != "/" || c.Domain != "" || c.MaxAge != 7200 {
		t.Fatalf("cookie: %+v", c)
	}
	if s := c.String(); !strings.Contains(s, "HttpOnly") || !strings.Contains(s, "Secure") || !strings.Contains(s, "SameSite=Strict") || strings.Contains(s, "Domain") {
		t.Fatalf("serialised cookie: %s", s)
	}
	if past := Cookie("tok", now.Add(-time.Hour), now); past.MaxAge != 1 {
		t.Fatalf("expired session cookie must still be short-lived: %+v", past)
	}
	cl := ClearCookie()
	if cl.MaxAge >= 0 || cl.Value != "" || !cl.Secure || !cl.HttpOnly {
		t.Fatalf("clear cookie: %+v", cl)
	}
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if _, ok := TokenFrom(r); ok {
		t.Fatal("no cookie")
	}
	r.AddCookie(&http.Cookie{Name: CookieName, Value: "abc"})
	if v, ok := TokenFrom(r); !ok || v != "abc" {
		t.Fatal("cookie read")
	}
}
