package owner

import (
	"errors"
	"testing"
)

func TestModuleValidation(t *testing.T) {
	for _, m := range []Module{Audit, FeatureFlags, "syn_alpha", "billing", "ab"} {
		if err := m.Validate(); err != nil {
			t.Fatalf("%q rejected: %v", m, err)
		}
	}
	for _, m := range []Module{"", "a", "Billing", "1billing", "bill-ing", "pg_catalog", "crm_x", "public", "information_schema", "a234567890123456789012345678901234567890xy"} {
		if err := m.Validate(); !errors.Is(err, ErrInvalidModule) {
			t.Fatalf("%q accepted: %v", m, err)
		}
	}
}

func TestSchemaAndRole(t *testing.T) {
	if Audit.Schema() != "audit" || Audit.Role() != "crm_owner_audit" {
		t.Fatalf("audit = %s / %s", Audit.Schema(), Audit.Role())
	}
	if FeatureFlags.Role() != "crm_owner_platform_featureflags" {
		t.Fatalf("flags role = %s", FeatureFlags.Role())
	}
	// Two modules never share a schema or a role.
	if Module("a1").Schema() == Module("a2").Schema() || Module("a1").Role() == Module("a2").Role() {
		t.Fatal("distinct modules must map to distinct schemas and roles")
	}
}
