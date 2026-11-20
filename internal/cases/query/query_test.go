package query_test

import (
	"context"
	"errors"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/store"
)

// RM-P01: customer_id is validated before any transaction opens (the
// service has no runner here) and never combines with an ambiguous link
// state, which would silently hide RELATED/UNKNOWN/CONFLICT rows.
func TestListCasesCustomerFilterValidation(t *testing.T) {
	s := query.New(nil)
	id := "a0000000-0000-4000-8000-00000000000a"
	str := func(v string) *string { return &v }
	for name, f := range map[string]store.CaseFilter{
		"not a uuid":      {CustomerID: str("not-a-uuid")},
		"phone":           {CustomerID: str("09120000000")},
		"legacy key":      {CustomerID: str("7001")},
		"uppercase":       {CustomerID: str("A0000000-0000-4000-8000-00000000000A")},
		"empty":           {CustomerID: str("")},
		"related":         {CustomerID: &id, CustomerLinkState: str("RELATED")},
		"unknown":         {CustomerID: &id, CustomerLinkState: str("UNKNOWN")},
		"conflict":        {CustomerID: &id, CustomerLinkState: str("CONFLICT")},
		"not recoverable": {CustomerID: &id, CustomerLinkState: str("NOT_RECOVERABLE")},
		"bad link state":  {CustomerID: &id, CustomerLinkState: str("MAYBE")},
	} {
		if _, err := s.ListCases(context.Background(), f, nil, nil); !errors.Is(err, query.ErrInvalidQuery) {
			t.Errorf("%s: %v", name, err)
		}
	}
}
