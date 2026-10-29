// Package owner names the write owner of a database transaction.
//
// Architecture V1 §5/§6/§8: one PostgreSQL schema per module, write grants on
// the module's own schema only, and one primary write owner per command. A
// Module value maps deterministically to its schema and to its NOLOGIN owner
// role. Every platform transaction runs under exactly one owner role
// (SET LOCAL ROLE), so a write into another module's schema is rejected by
// PostgreSQL itself.
package owner

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// ErrInvalidModule reports a malformed or reserved module name.
var ErrInvalidModule = errors.New("owner: invalid module name")

// RolePrefix prefixes every module owner role.
const RolePrefix = "crm_owner_"

// Module is a write-owner name. It is also the module's schema name.
type Module string

// Platform-owned modules created in Slice 1. Business modules are not
// declared here; each declares its own Module when it is implemented.
const (
	// Audit owns the append-only audit schema.
	Audit Module = "audit"
	// FeatureFlags owns the boolean feature-flag store.
	FeatureFlags Module = "platform_featureflags"
)

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{1,40}$`)

// Validate checks the module name. Names starting with "pg_" or "crm_" and
// the "public"/"information_schema" schemas are reserved.
func (m Module) Validate() error {
	s := string(m)
	if !namePattern.MatchString(s) || strings.HasPrefix(s, "pg_") || strings.HasPrefix(s, "crm_") ||
		s == "public" || s == "information_schema" {
		return fmt.Errorf("%w: %q", ErrInvalidModule, s)
	}
	return nil
}

// Schema returns the module's PostgreSQL schema name.
func (m Module) Schema() string { return string(m) }

// Role returns the module's NOLOGIN owner role name.
func (m Module) Role() string { return RolePrefix + string(m) }
