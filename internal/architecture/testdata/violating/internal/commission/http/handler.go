// Fixture: violates the Slice 13 read-path rules. Never compiled.
package http

// read-path-import: a GET never runs the offline shadow job.
import _ "example.test/mod/internal/commission/shadow"

// s13-read-write: no write SQL on the read path.
const write = "INSERT INTO commission.calculation (run_id) VALUES ($1)"
