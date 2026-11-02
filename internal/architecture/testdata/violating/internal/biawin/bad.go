// Fixture: violates the Slice 10 biawin rules. Never compiled.
package biawin

import (
	// module-boundary: biawin never reaches billing internals.
	_ "example.test/mod/internal/billing/query"
	// biawin-provider-call: no provider is wired in Slice 10.
	_ "example.test/mod/internal/integration/biawin"
)

// foreign-schema-write + append-only-write + biawin-foreign-sql: biawin
// never mutates finance (or billing / payments).
const mutateFinance = "UPDATE finance.ledger_entry SET amount = 0"

// biawin-foreign-sql + phone-key: no foreign read and no phone join.
const phoneJoin = "SELECT * FROM billing.invoice i JOIN biawin.subscription s ON s.customer_phone = i.phone"

// append-only-write: subscription history is never rewritten.
const rewrite = "DELETE FROM biawin.subscription WHERE run_id = $1"

// biawin-provider-call (literal).
const op = "charge_wallet"

// biawin-provider-call + biawin-forbidden-command (identifier).
func ChargeWallet() {}

// biawin-forbidden-command: no activation, Dot completion, membership
// write or commission execution.
func ActivateSubscription()   {}
func CompleteDot()            {}
func WriteProjectMembership() {}
func RunCommission()          {}

// phone-key (identifier).
var byPhone string
