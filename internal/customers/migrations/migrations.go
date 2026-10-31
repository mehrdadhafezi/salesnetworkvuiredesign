// Package migrations holds the customers module's versioned DDL (Architecture
// V1 §8.11). The files are applied only by cmd/crm-migrate in their own
// version table; nothing applies them on application start.
//
// 00001 is rendered with the platform module template (schema, NOLOGIN owner
// role, per-module outbox/inbox/idempotency) followed by body/00001_customers.sql.
// The committed file is checked byte-for-byte by this package's tests;
// regenerate with
//
//	go test ./internal/customers/migrations -run TestRenderedMigration -args -update
package migrations

import (
	"embed"
	"io/fs"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
)

// Table records applied customers migrations.
const Table = "crm_schema_migrations_customers"

//go:embed sql/*.sql
var files embed.FS

// Source returns the embedded customers migration source.
func Source() migrations.Source {
	sub, err := fs.Sub(files, "sql")
	if err != nil {
		panic(err) // embedded at build time; cannot fail
	}
	return migrations.Source{FS: sub, Table: Table}
}
