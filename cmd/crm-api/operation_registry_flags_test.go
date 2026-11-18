package main

import (
	"bytes"
	"os"
	"testing"

	"gopkg.in/yaml.v3"

	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
)

// RM-E04B / A4 (NT-22): the feature flag of every entry of the operation
// scope registry equals the flag the composed guard really uses for its
// gate capabilities (capabilityFlags), and the session commands use the
// session-auth flag. The registry is validated in full by
// internal/authorization/http/operation_registry_test.go; this test covers
// the one field that only this package can see.

const operationRegistry = "../../internal/authorization/http/testdata/operation_scope_registry.yaml"

func TestOperationRegistryFeatureFlags(t *testing.T) {
	b, err := os.ReadFile(operationRegistry)
	if err != nil {
		t.Fatal(err)
	}
	var reg struct {
		Operations []struct {
			OperationID  string   `yaml:"operation_id"`
			Category     string   `yaml:"category"`
			Capability   string   `yaml:"capability"`
			AlsoRequires []string `yaml:"also_requires"`
			FeatureFlag  string   `yaml:"feature_flag"`
		} `yaml:"operations"`
	}
	dec := yaml.NewDecoder(bytes.NewReader(b))
	if err := dec.Decode(&reg); err != nil {
		t.Fatal(err)
	}
	if len(reg.Operations) != 105 {
		t.Fatalf("registry has %d entries, want 105", len(reg.Operations))
	}
	for _, e := range reg.Operations {
		switch {
		case e.Capability != "":
			for _, c := range append([]string{e.Capability}, e.AlsoRequires...) {
				want, ok := capabilityFlags[identityapi.Capability(c)]
				if !ok {
					t.Errorf("%s: gate capability %s has no guard flag", e.OperationID, c)
				} else if e.FeatureFlag != want {
					t.Errorf("%s: registry feature_flag %q, guard flag of %s is %q", e.OperationID, e.FeatureFlag, c, want)
				}
			}
		case e.Category == "SESSION_COMMAND":
			if e.FeatureFlag != SessionAuthFlag {
				t.Errorf("%s: registry feature_flag %q, want %q", e.OperationID, e.FeatureFlag, SessionAuthFlag)
			}
		default:
			if e.FeatureFlag != "" {
				t.Errorf("%s: capability-free operation declares feature_flag %q", e.OperationID, e.FeatureFlag)
			}
		}
	}
}
