// Fixture: violates the Slice 10 read-path rules. Never compiled.
package http

// read-path-import: a GET never imports.
import _ "example.test/mod/internal/biawin/imports"

// biawin-read-write: no write SQL on the read path.
const write = "INSERT INTO biawin.subscription (run_id) VALUES ($1)"

// biawin-forbidden-command: no activation route.
const route = "/api/v1/biawin/subscriptions/{subscription_id}/activate"
