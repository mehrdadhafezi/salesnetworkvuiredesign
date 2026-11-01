// Fixture: allowed. Aggregates live only in the shared query spec, over
// reporting tables only. Never compiled.
package query

const summary = "SELECT count(*) FROM reporting.case_fact f WHERE f.run_id = $1"
