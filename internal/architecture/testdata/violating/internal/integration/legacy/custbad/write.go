package custbad

// Violates foreign-schema-write and module-boundary (customers store).
import _ "example.test/mod/internal/customers/store"

const q = "UPDATE customers.customer SET customer_id = $1"
