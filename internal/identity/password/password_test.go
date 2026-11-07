package password

import (
	"errors"
	"strings"
	"testing"
)

var fast = Params{MemoryKiB: 64, Iterations: 1, Parallelism: 1, SaltBytes: 16, KeyBytes: 32}

func TestHashVerify(t *testing.T) {
	h, err := Hash("synthetic correct horse", fast)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(h, "$argon2id$v=19$m=64,t=1,p=1$") {
		t.Fatalf("phc = %s", h)
	}
	if ok, err := Verify(h, "synthetic correct horse"); !ok || err != nil {
		t.Fatalf("verify: %v %v", ok, err)
	}
	for _, wrong := range []string{"synthetic correct hors", "Synthetic correct horse", "", strings.Repeat("x", MaxPasswordBytes+1)} {
		if ok, _ := Verify(h, wrong); ok {
			t.Fatalf("%q verified", wrong)
		}
	}
	h2, _ := Hash("synthetic correct horse", fast)
	if h2 == h {
		t.Fatal("salt is not random")
	}
}

func TestDefaultParams(t *testing.T) {
	h, err := Hash("synthetic", Default)
	if err != nil || !strings.HasPrefix(h, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Fatalf("%s %v", h, err)
	}
}

// Legacy WordPress / phpass / bcrypt / MD5 hashes are never accepted, and a
// tampered cost parameter cannot exhaust memory.
func TestRejectsForeignAndUnboundedHashes(t *testing.T) {
	for _, phc := range []string{
		"$P$BXr3jM7S0QnXq1M0j5bkxGJ0cV8F2c/",                                             // WordPress phpass
		"$wp$2y$10$abcdefghijklmnopqrstuuK3k2rGfX8y8n3u3Q1y1QhGvGq4xQ6Jm",                // WP 6.8 bcrypt
		"$2y$10$abcdefghijklmnopqrstuuK3k2rGfX8y8n3u3Q1y1QhGvGq4xQ6Jm",                   // bcrypt
		"5f4dcc3b5aa765d61d8327deb882cf99",                                               // md5
		"$argon2i$v=19$m=64,t=1,p=1$c29tZXNhbHRzb21lc2FsdA$aGFzaGhhc2hoYXNoaGFzaA",       // argon2i
		"$argon2id$v=19$m=9999999,t=1,p=1$c29tZXNhbHRzb21lc2FsdA$aGFzaGhhc2hoYXNoaGFzaA", // memory bomb
		"$argon2id$v=19$m=64,t=99,p=1$c29tZXNhbHRzb21lc2FsdA$aGFzaGhhc2hoYXNoaGFzaA",     // iteration bomb
		"$argon2id$v=19$m=64,t=1,p=1,x=1$c29tZXNhbHRzb21lc2FsdA$aGFzaGhhc2hoYXNoaGFzaA",  // unknown param
		"$argon2id$v=16$m=64,t=1,p=1$c29tZXNhbHRzb21lc2FsdA$aGFzaGhhc2hoYXNoaGFzaA",      // old version
	} {
		ok, err := Verify(phc, "password")
		if ok || !errors.Is(err, ErrMalformed) {
			t.Errorf("%s: ok=%v err=%v", phc, ok, err)
		}
	}
}
