// Package review is a VIOLATING Slice 12 fixture.
package review

import "example.test/mod/internal/platform/db"

var _ db.Tx

const q = `SELECT * FROM payments.submitted_attempt`
