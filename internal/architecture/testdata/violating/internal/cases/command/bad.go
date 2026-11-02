// Fixture: violates the Slice 11 command rules. Never compiled.
package command

import (
	// s11-financial-import + module-boundary: cases never imports billing.
	_ "example.test/mod/internal/billing/api"
	// s11-domain-connector: domains never call the write-back worker.
	_ "example.test/mod/internal/integration/writeback"
)

// s11-outbox-required: a command write without an outbox append.
const bump = "UPDATE cases.work_case SET version = version + 1 WHERE case_id = $1"

// s11-forbidden-command: ADR-01 / F01 / unproven paths stay unavailable.
func IssueInvoice() {}
func ReturnCase()   {}
func MarkReady()    {}

// s11-rank-shortcut and s11-unguarded-bulk.
func IsHigherRank() bool { return true }
func BulkReassign()      {}

// phone-key: never a command identity.
var byPhone string
