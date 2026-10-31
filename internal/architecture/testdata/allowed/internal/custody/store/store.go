package store

// Allowed: custody appends its own events (INSERT only).
const appendEvent = "INSERT INTO custody.custody_event (event_id) VALUES ($1)"
