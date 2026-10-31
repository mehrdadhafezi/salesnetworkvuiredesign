package shadowbad

// Violations: the connector imports a module command implementation and
// writes a target schema directly.
import _ "example.test/mod/internal/organization/imports"

const write = "INSERT INTO organization.person (person_id) VALUES ($1)"
