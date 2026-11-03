// Package command is an ALLOWED Slice 12 fixture.
package command

import "net/http"

// SubmitBatch is the permitted owning-seller batch command.
func SubmitBatch() {}

const insert = `INSERT INTO payments.submitted_attempt (attempt_id, state) VALUES ($1, 'PENDING_REVIEW')`

const route = "/api/v1/payments/invoices/{invoice_id}/batches"

func media(b []byte) string { return http.DetectContentType(b) }
