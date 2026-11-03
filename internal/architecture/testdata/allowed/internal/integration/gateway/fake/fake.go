// Package fake is an ALLOWED Slice 12 fixture: an in-memory provider.
package fake

// Request is idempotent per request id; no network.
func Request(id string) string { return "syn-" + id }
