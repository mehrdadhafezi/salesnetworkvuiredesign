// Package migrations holds the integration module's versioned staging DDL
// (Architecture V1 §8.11, §24.3). The files are applied only by
// cmd/crm-migrate, in their own version table; nothing applies them on
// application start, and the legacy import tool never runs DDL.
//
// 00001 is rendered with the platform module template (schema, NOLOGIN owner
// role, per-module outbox/inbox/idempotency) followed by the import batch
// tables and one raw staging table per published extract spec. The
// committed file is checked byte-for-byte against the template and the specs
// by this package's tests; regenerate with
//
//	go test ./internal/integration/legacy/migrations -run TestRenderedMigration -update
//
// Rollback is "drop the staging schema" (the migration's Down section) on a
// disposable or development database only.
package migrations

import (
	"embed"
	"io/fs"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
)

// Table records applied integration migrations, separately from the
// platform's version table.
const Table = "crm_schema_migrations_integration"

//go:embed sql/*.sql
var files embed.FS

// Source returns the embedded integration migration source.
func Source() migrations.Source {
	sub, err := fs.Sub(files, "sql")
	if err != nil {
		panic(err) // embedded at build time; cannot fail
	}
	return migrations.Source{FS: sub, Table: Table}
}
