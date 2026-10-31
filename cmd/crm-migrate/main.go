// Command crm-migrate is the only way target DDL reaches a database
// (Architecture V1 §8.11). It applies the embedded platform migrations, then
// the integration staging migrations (Slice 3), then the identity and
// organization read-model migrations (Slice 4), then the customers identity
// shadow migrations (Slice 5), each source in its own version table; no application process runs migrations on startup.
//
//	CRM_MIGRATE_DATABASE_URL=postgres://... crm-migrate up
//	CRM_MIGRATE_DATABASE_URL=postgres://... crm-migrate status
//
// Exit status is non-zero on any failure; the run stops at the first failing
// migration and that migration is not recorded as applied.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	customersmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers/migrations"
	identitymigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/migrations"
	integrationmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/migrations"
	organizationmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
)

// sources returns every migration source in apply order: platform first.
func sources() []migrations.Source {
	return []migrations.Source{migrations.Platform(), integrationmigrations.Source(),
		identitymigrations.Source(), organizationmigrations.Source(), customersmigrations.Source()}
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(run(ctx, os.Args[1:], os.Getenv, os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	if len(args) != 1 || (args[0] != "up" && args[0] != "status") {
		fmt.Fprintln(stderr, "usage: crm-migrate up|status   (DSN from "+migrations.EnvMigrateURL+")")
		return 2
	}
	dsn := getenv(migrations.EnvMigrateURL)
	if dsn == "" {
		fmt.Fprintln(stderr, "crm-migrate: "+migrations.EnvMigrateURL+" is not set")
		return 2
	}
	switch args[0] {
	case "up":
		total := 0
		for _, src := range sources() {
			applied, err := migrations.Up(ctx, dsn, src)
			for _, a := range applied {
				fmt.Fprintf(stdout, "applied %05d %s\n", a.Version, a.Name)
			}
			total += len(applied)
			if err != nil {
				fmt.Fprintln(stderr, "crm-migrate:", err)
				return 1
			}
		}
		if total == 0 {
			fmt.Fprintln(stdout, "no pending migrations")
		}
	case "status":
		for _, src := range sources() {
			states, err := migrations.Status(ctx, dsn, src)
			if err != nil {
				fmt.Fprintln(stderr, "crm-migrate:", err)
				return 1
			}
			for _, s := range states {
				state := "pending"
				if s.Applied {
					state = "applied"
				}
				fmt.Fprintf(stdout, "%05d %-8s %s\n", s.Version, state, s.Name)
			}
		}
	}
	return 0
}
