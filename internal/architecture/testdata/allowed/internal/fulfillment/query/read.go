// Fixture: the fulfillment read path. Never compiled.
package query

const read = "SELECT case_id FROM fulfillment.fulfillment_case WHERE run_id = $1"

const route = "/api/v1/fulfillment/cases/{case_id}/history"
