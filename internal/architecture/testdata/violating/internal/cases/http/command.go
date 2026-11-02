// Fixture: violates the Slice 11 http rules. Never compiled.
package http

// s11-domain-connector + legacy-import: no connector from a handler.
import _ "example.test/mod/internal/integration/legacy/writeback"

// s11-read-path-sql: handlers hold no SQL.
const read = "SELECT version FROM cases.work_case WHERE case_id = $1"

// s11-forbidden-command: no ordinary return route (ADR-01 / F01 / OPD-03).
const route = "/api/v1/cases/{case_id}/return"
