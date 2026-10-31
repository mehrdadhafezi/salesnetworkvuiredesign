package store

// Allowed: customers writes its own schema.
const insertCustomer = "INSERT INTO customers.customer (customer_id) VALUES ($1)"

// MergeCaseState is a read-only candidate state, not a merge operation.
type MergeCaseState string
