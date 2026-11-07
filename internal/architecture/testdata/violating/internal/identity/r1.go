// Fixture: violates the R1 credential rule. Never compiled.
package identity

// r1-credential-write: no production code enrols or resets a credential.
const enrol = "INSERT INTO identity.credential (credential_id, user_id, login_name, kind, secret_hash, status) VALUES ($1, $2, $3, 'PASSWORD_ARGON2ID', $4, 'ACTIVE')"
const reset = "UPDATE identity.credential SET secret_hash = $1 WHERE user_id = $2"

// allowed: reading credentials, and writing sessions.
const read = "SELECT secret_hash FROM identity.credential WHERE login_name = $1"
const session = "INSERT INTO identity.session (session_id) VALUES ($1)"
