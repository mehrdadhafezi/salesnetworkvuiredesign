// Package command is a VIOLATING Slice 12 fixture.
package command

import (
	"net/http"
	"os"
)

// ApproveAttempt is a forbidden Finance decision (D-41 / OPD-08).
func ApproveAttempt() {}

// IssueNextStage is a forbidden billing transition (D-32).
func IssueNextStage() {}

// AutoLogin would issue a session after a payment (D-26).
func AutoLogin() {}

const route = "/api/v1/payments/attempts/{attempt_id}/approve"

const accept = `UPDATE payments.submitted_attempt SET state = 'ACCEPTED' WHERE attempt_id = $1`

func call() {
	_, _ = http.Get("https://gateway.invalid/pay")
	_, _ = os.Open("/var/receipts/a.pdf")
}
