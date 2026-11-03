// Fixture: violates the Slice 13 commission rules. Never compiled.
package commission

import "example.test/mod/internal/platform/outbox"

// s13-forbidden-command: approval and posting are decision-blocked.
func ApproveRun()     {}
func PostCommission() {}

const approveRoute = "/api/v1/commission/runs/{run_id}/approve"

// s13-float: amounts are exact scaled integers.
var amount float64

// s13-posting-dispatch: a posting request is never dispatched.
func dispatch() { outbox.Append(nil) }

// s13-foreign-sql: commission never reads the current hierarchy.
const hierarchy = "SELECT parent_id FROM organization.assignment"
