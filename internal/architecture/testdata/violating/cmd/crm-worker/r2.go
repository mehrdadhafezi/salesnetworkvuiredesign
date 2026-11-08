// Fixture: violates the R2 worker rules. Never compiled.
package main

import "net/http"

// r2-network-call: the worker binary never opens an external connection.
func call() { _, _ = http.Get("https://legacy.example.test/wp-admin/admin-ajax.php") }

// r2-worker-sql: the worker binary holds no SQL.
const write = "INSERT INTO payments.receipt_evidence (receipt_id) VALUES ($1)"
const read = "SELECT state FROM integration.writeback_command WHERE command_id = $1"

// allowed: plain configuration strings.
const job = "receipt_finalise"
