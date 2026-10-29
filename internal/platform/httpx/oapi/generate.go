// Package oapi holds the Go transport types and strict server interface
// generated from the authoritative OpenAPI contract
// (api/openapi/openapi.bundled.yaml, bundled from api/openapi/openapi.yaml).
//
// Never edit oapi.gen.go by hand: run `npm --prefix web run openapi:gen`.
// CI regenerates and fails when the committed output is stale.
package oapi

//go:generate go tool oapi-codegen -config oapi-codegen.yaml ../../../../api/openapi/openapi.bundled.yaml
