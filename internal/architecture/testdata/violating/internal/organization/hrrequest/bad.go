// Fixture: violates the Slice 11 HR rules. Never compiled.
package hrrequest

// s11-hr-identity-write: HR never grants capabilities (D-42).
func GrantRole() {}

// s11-rank-shortcut: no administrator shortcut.
const admin = "manage_options"

// s11-forbidden-command: HR final apply is blocked (OPD-05 / SD-04).
func ApplyHRRequest() {}
