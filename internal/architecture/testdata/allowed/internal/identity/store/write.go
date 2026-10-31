package store

// Allowed: identity writes its own schema.
const write = "INSERT INTO identity.user_account (user_id) VALUES ($1)"
