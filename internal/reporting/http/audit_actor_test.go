package http

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/refs"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/audit"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/websession"
)

// R1: a sensitive read by a real session records the authenticated actor
// (identifiers only), never the session or CSRF token.
func TestAuditRecordsSessionActor(t *testing.T) {
	at, _ := timeutil.NewInstant(time.Date(2026, 11, 7, 9, 0, 0, 0, time.UTC))
	var got audit.Event
	h := &Handlers{clock: timeutil.FixedClock{At: at}, auditor: func(_ context.Context, e audit.Event) error { got = e; return nil }}
	token, _ := websession.NewToken()
	p := websession.Principal{SessionID: "33333333-3333-4333-8333-333333333333", UserID: "11111111-1111-4111-8111-111111111111", CSRF: websession.CSRFToken(token)}
	ctx := websession.WithPrincipal(context.Background(), p)
	subject, err := refs.NewNativeRef(refs.DomainInvoice, "1")
	if err != nil {
		t.Fatal(err)
	}
	if err := h.auditRead(ctx, subject, map[string]string{"surface": "invoice_review.detail"}); err != nil {
		t.Fatal(err)
	}
	if got.Metadata["user_id"] != p.UserID || got.Metadata["session_id"] != p.SessionID || got.Metadata["principal"] != "" {
		t.Fatalf("actor metadata: %v", got.Metadata)
	}
	for k, v := range got.Metadata {
		if strings.Contains(v, token) || strings.Contains(v, p.CSRF) {
			t.Fatalf("secret recorded in %s", k)
		}
	}
}
