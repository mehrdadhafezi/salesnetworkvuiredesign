// Fixture: a per-unit finance read. Never compiled.
package query

const perUnit = "SELECT l.unit, sum(l.amount) FROM finance.ledger_line l WHERE l.run_id = $1 GROUP BY l.unit"
