// Fixture: violates status-fallback (twice) and module-boundary. Never compiled.
package query

import _ "example.test/mod/internal/payments/api"

const display = "SELECT COALESCE(NULLIF(status,''), NULLIF(invoice_status,''), '') FROM billing.invoice"

func resolveMainStatus() string { return display }
