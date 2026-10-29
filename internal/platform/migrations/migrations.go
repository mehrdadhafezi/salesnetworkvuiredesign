// Package migrations is the only DDL path of the target system
// (Architecture V1 §8.11). Migrations are versioned SQL files applied
// explicitly by cmd/crm-migrate; nothing applies them on application start.
//
// Mechanism: pressly/goose v3 Provider with a PostgreSQL advisory session lock
// (one migrator at a time). Each migration file runs in its own transaction,
// so a failing migration rolls back completely and is never recorded as
// applied; the run stops at the first failure. "NO TRANSACTION" migrations are
// rejected by Validate.
package migrations

import (
	"bytes"
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strings"
	"text/template"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
)

// EnvMigrateURL holds the migrator DSN. It is distinct from the application
// DSN: the migrator owns DDL, the application does not.
const EnvMigrateURL = "CRM_MIGRATE_DATABASE_URL"

// PlatformTable records applied platform migrations.
const PlatformTable = "crm_schema_migrations"

var (
	// ErrInvalidMigration reports a migration file that breaks the rules.
	ErrInvalidMigration = errors.New("migrations: invalid migration")
	// ErrInvalidSource reports an unusable migration source.
	ErrInvalidSource = errors.New("migrations: invalid source")
)

//go:embed sql/*.sql
var platformFS embed.FS

//go:embed templates/module_infra.sql.tmpl
var moduleInfraTemplate string

// Source is a set of migration files tracked in its own version table.
type Source struct {
	FS    fs.FS
	Table string
}

// Platform returns the embedded platform migration source.
func Platform() Source {
	sub, err := fs.Sub(platformFS, "sql")
	if err != nil {
		panic(err) // embedded at build time; cannot fail
	}
	return Source{FS: sub, Table: PlatformTable}
}

var (
	fileName      = regexp.MustCompile(`^[0-9]{5}_[a-z0-9_]+\.sql$`)
	tableName     = regexp.MustCompile(`^[a-z][a-z0-9_]{2,62}$`)
	noTxDirective = regexp.MustCompile(`(?m)^\s*--\s*\+goose\s+NO\s+TRANSACTION`)
	upDirective   = regexp.MustCompile(`(?m)^\s*--\s*\+goose\s+Up\s*$`)
)

// Validate checks every file of src: deterministic five-digit versions,
// unique versions, an Up section, and no NO TRANSACTION directive.
func Validate(src Source) ([]string, error) {
	if src.FS == nil || !tableName.MatchString(src.Table) {
		return nil, ErrInvalidSource
	}
	entries, err := fs.ReadDir(src.FS, ".")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidSource, err)
	}
	var names []string
	versions := map[string]string{}
	for _, e := range entries {
		if e.IsDir() {
			return nil, fmt.Errorf("%w: unexpected directory %s", ErrInvalidMigration, e.Name())
		}
		name := e.Name()
		if !fileName.MatchString(name) {
			return nil, fmt.Errorf("%w: file name %q must be NNNNN_name.sql", ErrInvalidMigration, name)
		}
		if prev, dup := versions[name[:5]]; dup {
			return nil, fmt.Errorf("%w: version %s used by %s and %s", ErrInvalidMigration, name[:5], prev, name)
		}
		versions[name[:5]] = name
		body, err := fs.ReadFile(src.FS, name)
		if err != nil {
			return nil, err
		}
		if !upDirective.Match(body) {
			return nil, fmt.Errorf("%w: %s has no -- +goose Up section", ErrInvalidMigration, name)
		}
		if noTxDirective.Match(body) {
			return nil, fmt.Errorf("%w: %s uses NO TRANSACTION; every migration must be atomic", ErrInvalidMigration, name)
		}
		names = append(names, name)
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("%w: no migrations", ErrInvalidSource)
	}
	sort.Strings(names)
	return names, nil
}

// Applied is one migration applied by Up.
type Applied struct {
	Version int64
	Name    string
}

// Up applies every pending migration of src in version order. It stops at
// the first failure and returns the migrations applied before it together
// with the error. Running Up again when nothing is pending is a no-op.
func Up(ctx context.Context, dsn string, src Source) ([]Applied, error) {
	p, closeDB, err := provider(dsn, src)
	if err != nil {
		return nil, err
	}
	defer closeDB()
	results, err := p.Up(ctx)
	var partial *goose.PartialError
	if errors.As(err, &partial) {
		results = partial.Applied // migrations committed before the failing one
	}
	var out []Applied
	for _, r := range results {
		if r.Error == nil && r.Source != nil {
			out = append(out, Applied{Version: r.Source.Version, Name: baseName(r.Source.Path)})
		}
	}
	if err != nil {
		return out, fmt.Errorf("migrations: up stopped: %w", err)
	}
	return out, nil
}

// State is one migration's status.
type State struct {
	Version int64
	Name    string
	Applied bool
}

// Status reports every migration of src and whether it is applied.
func Status(ctx context.Context, dsn string, src Source) ([]State, error) {
	p, closeDB, err := provider(dsn, src)
	if err != nil {
		return nil, err
	}
	defer closeDB()
	rows, err := p.Status(ctx)
	if err != nil {
		return nil, fmt.Errorf("migrations: status: %w", err)
	}
	out := make([]State, 0, len(rows))
	for _, r := range rows {
		out = append(out, State{Version: r.Source.Version, Name: baseName(r.Source.Path), Applied: r.State == goose.StateApplied})
	}
	return out, nil
}

func provider(dsn string, src Source) (*goose.Provider, func(), error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, nil, fmt.Errorf("%w: empty DSN", ErrInvalidSource)
	}
	if _, err := Validate(src); err != nil {
		return nil, nil, err
	}
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("migrations: open: %w", err)
	}
	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		_ = sqlDB.Close()
		return nil, nil, err
	}
	p, err := goose.NewProvider(goose.DialectPostgres, sqlDB, src.FS,
		goose.WithTableName(src.Table),
		goose.WithSessionLocker(locker),
		goose.WithDisableGlobalRegistry(true),
	)
	if err != nil {
		_ = sqlDB.Close()
		return nil, nil, fmt.Errorf("migrations: provider: %w", err)
	}
	return p, func() { _ = p.Close() }, nil
}

func baseName(path string) string {
	if i := strings.LastIndexAny(path, `/\`); i >= 0 {
		return path[i+1:]
	}
	return path
}

// RenderModuleInfra renders the complete migration that creates module m's
// schema, owner role and its own outbox, inbox and idempotency tables,
// followed by moduleUp (the module's own table DDL) and a Down section.
// Every business module's first migration is produced this way, so that
// infrastructure tables are always per-module and never shared.
func RenderModuleInfra(m owner.Module, moduleUp string) (string, error) {
	if err := m.Validate(); err != nil {
		return "", err
	}
	if strings.Contains(moduleUp, "+goose") {
		return "", fmt.Errorf("%w: module SQL must not contain goose directives", ErrInvalidMigration)
	}
	t, err := template.New("module_infra").Option("missingkey=error").Parse(moduleInfraTemplate)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	data := struct{ Schema, Role, ModuleUp string }{m.Schema(), m.Role(), strings.TrimSpace(moduleUp)}
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
