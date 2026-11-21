package query_test

import (
	"context"
	"errors"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/store"
)

// RM-P01: customer_id / case_id are validated before any transaction opens
// (the service has no runner here) and never combine with an ambiguous link
// state of the same relationship.
func TestListInvoicesLinkFilterValidation(t *testing.T) {
	s := query.New(nil)
	cu, ca := "a0000000-0000-4000-8000-00000000000a", "c0000000-0000-4000-8000-0000000000a1"
	str := func(v string) *string { return &v }
	for name, f := range map[string]store.InvoiceFilter{
		"customer not uuid":      {CustomerID: str("not-a-uuid")},
		"customer phone":         {CustomerID: str("09120000000")},
		"case legacy key":        {CaseID: str("7001")},
		"case empty":             {CaseID: str("")},
		"customer + RELATED":     {CustomerID: &cu, CustomerLink: str("RELATED")},
		"customer + CONFLICT":    {CustomerID: &cu, CustomerLink: str("CONFLICT")},
		"customer + UNKNOWN":     {CustomerID: &cu, CustomerLink: str("UNKNOWN")},
		"case + NOT_RECOVERABLE": {CaseID: &ca, CaseLink: str("NOT_RECOVERABLE")},
		"case + CONFLICT":        {CaseID: &ca, CaseLink: str("CONFLICT")},
		"both, case ambiguous":   {CustomerID: &cu, CaseID: &ca, CaseLink: str("RELATED")},
	} {
		if _, err := s.ListInvoices(context.Background(), f, nil, nil); !errors.Is(err, query.ErrInvalidQuery) {
			t.Errorf("%s: %v", name, err)
		}
	}
}
