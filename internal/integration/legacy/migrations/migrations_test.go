package migrations

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/spec"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
)

var update = flag.Bool("update", false, "rewrite the rendered integration migration")

const (
	bodyFile     = "body/00001_integration_staging.sql"
	renderedFile = "sql/00001_integration_staging.sql"
	genBegin     = "-- BEGIN GENERATED STAGING TABLES (do not edit; see package doc)"
	genEnd       = "-- END GENERATED STAGING TABLES"
)

// stagingDDL renders the raw staging table of one spec. It lives in a test
// file on purpose: production code never builds DDL (architecture lint).
func stagingDDL(s spec.Spec) string {
	t := "integration." + s.StagingTable
	var b strings.Builder
	fmt.Fprintf(&b, "\n-- %s: raw %s (%d columns; classification %s)\n", s.ID, s.SourceTable, len(s.Columns), s.Classification())
	fmt.Fprintf(&b, "CREATE TABLE %s (\n", t)
	b.WriteString("    batch_id       uuid        NOT NULL,\n")
	b.WriteString("    spec_id        text        NOT NULL,\n")
	b.WriteString("    source_ordinal bigint      NOT NULL,\n")
	b.WriteString("    source_key     text        NOT NULL,\n")
	b.WriteString("    row_sha256     bytea       NOT NULL,\n")
	b.WriteString("    imported_at    timestamptz NOT NULL,\n")
	for _, c := range s.Columns {
		fmt.Fprintf(&b, "    %s text NULL,\n", c.StagingColumn())
	}
	b.WriteString("    PRIMARY KEY (batch_id, source_ordinal),\n")
	fmt.Fprintf(&b, "    CONSTRAINT %s_key_unique UNIQUE (batch_id, source_key),\n", s.StagingTable)
	fmt.Fprintf(&b, "    CONSTRAINT %s_spec CHECK (spec_id = '%s'),\n", s.StagingTable, s.ID)
	fmt.Fprintf(&b, "    CONSTRAINT %s_ordinal CHECK (source_ordinal > 0),\n", s.StagingTable)
	fmt.Fprintf(&b, "    CONSTRAINT %s_row_sha CHECK (octet_length(row_sha256) = 32),\n", s.StagingTable)
	fmt.Fprintf(&b, "    CONSTRAINT %s_entity_fk FOREIGN KEY (batch_id, spec_id) REFERENCES integration.import_entity (batch_id, spec_id)\n", s.StagingTable)
	b.WriteString(");\n")
	fmt.Fprintf(&b, "GRANT SELECT, INSERT ON %s TO %s;\n", t, integration.Module.Role())
	return b.String()
}

// slice3Specs returns the Slice 3 published set in registry (ID) order: the
// tables of 00001. Later published sets get their own migrations.
func slice3Specs(t *testing.T) []spec.Spec {
	t.Helper()
	r, err := spec.NewRegistry(spec.V1Slice3()...)
	if err != nil {
		t.Fatal(err)
	}
	return r.Specs()
}

const slice5File = "sql/00003_integration_staging_invoice_issued_to.sql"

// renderSlice5 renders the Slice 5 staging migration: the raw staging table
// of every Slice 5 spec, generated exactly like 00001's tables.
func renderSlice5(t *testing.T) string {
	t.Helper()
	r, err := spec.NewRegistry(spec.V1Slice5()...)
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	b.WriteString("-- Integration raw staging, Slice 5 extension: invoice ISSUED-TO identity\n")
	b.WriteString("-- evidence only (Architecture V1 §10, §24.3). Generated from the published\n")
	b.WriteString("-- Slice 5 specs; regenerate with the package -update flag.\n\n")
	b.WriteString("-- +goose Up\n")
	var down []string
	for _, s := range r.Specs() {
		b.WriteString(stagingDDL(s))
		down = append(down, "DROP TABLE integration."+s.StagingTable+";\n")
	}
	b.WriteString("\n-- +goose Down\n")
	b.WriteString(strings.Join(down, ""))
	return b.String()
}

// TestRenderedSliceFiveMigration proves 00003 is exactly the generated DDL.
func TestRenderedSliceFiveMigration(t *testing.T) {
	want := renderSlice5(t)
	if *update {
		if err := os.WriteFile(slice5File, []byte(want), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := os.ReadFile(slice5File)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("%s is stale: regenerate with -update", slice5File)
	}
}

func renderExpected(t *testing.T) (body, rendered string) {
	t.Helper()
	raw, err := os.ReadFile(bodyFile)
	if err != nil {
		t.Fatal(err)
	}
	src := string(raw)
	i, j := strings.Index(src, genBegin), strings.Index(src, genEnd)
	if i < 0 || j < i {
		t.Fatalf("%s: generated-section markers missing", bodyFile)
	}
	var gen strings.Builder
	for _, s := range slice3Specs(t) {
		gen.WriteString(stagingDDL(s))
	}
	body = src[:i+len(genBegin)] + "\n" + gen.String() + src[j:]
	out, err := migrations.RenderModuleInfra(integration.Module, body)
	if err != nil {
		t.Fatal(err)
	}
	return body, out
}

// TestRenderedMigration proves the committed migration is exactly the module
// template applied to the body, whose staging tables are generated from the
// published specs. A spec change without a new migration fails here.
func TestRenderedMigration(t *testing.T) {
	body, want := renderExpected(t)
	if *update {
		if err := os.WriteFile(bodyFile, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(renderedFile, []byte(want), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := os.ReadFile(renderedFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("%s is stale: regenerate with -update", renderedFile)
	}
	if b, _ := os.ReadFile(bodyFile); string(b) != body {
		t.Fatalf("%s generated section is stale: regenerate with -update", bodyFile)
	}
}

func TestSourceValidates(t *testing.T) {
	names, err := migrations.Validate(Source())
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 3 || names[0] != filepath.Base(renderedFile) || names[1] != "00002_integration_staging_guards.sql" ||
		names[2] != filepath.Base(slice5File) {
		t.Fatalf("migration files = %v", names)
	}
	if Source().Table == migrations.PlatformTable {
		t.Fatal("integration must use its own version table")
	}
}
