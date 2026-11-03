// Package review is an ALLOWED Slice 12 fixture: a pure evaluator.
package review

// ActionApprove names a gated (never allowed) action.
const ActionApprove = "finance.attempt.approve"

// Evaluate is pure.
func Evaluate(ready bool) bool { return !ready && false }
