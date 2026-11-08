// Fixture: violates the R2 network rule. Never compiled.
package writeback

import "net/http"

// r2-network-call: only the compat transport may build a request.
func send() { _, _ = http.NewRequest("POST", "https://legacy.example.test/x", nil) }
