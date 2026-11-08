// Fixture: the compat transport is the one place allowed to build requests.
package compat

import "net/http"

func send() { _, _ = http.NewRequest("POST", "https://legacy.example.test/x", nil) }
