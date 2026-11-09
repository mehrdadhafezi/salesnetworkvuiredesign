package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthcheck(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	defer ok.Close()
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(503) }))
	defer bad.Close()
	if run([]string{ok.URL + "/health/ready"}, io.Discard) != 0 || run([]string{bad.URL}, io.Discard) != 1 {
		t.Fatal("status mapping")
	}
	for _, u := range []string{"https://127.0.0.1/x", "http://example.com/health", "http://10.0.0.5/health", "http://u:p@127.0.0.1/x", "ftp://127.0.0.1"} {
		if run([]string{u}, io.Discard) != 2 {
			t.Fatalf("%s accepted", u)
		}
	}
	if run([]string{"http://127.0.0.1:1/closed"}, io.Discard) != 1 || run(nil, io.Discard) != 2 {
		t.Fatal("unavailable / usage")
	}
}
