package spec

// Extraction-scope guard (Slice 3).
//
// Some published specs describe a source table whose ROW SCOPE is an
// unresolved Owner decision. The spec defines columns and raw semantics only;
// it does not authorise extracting every row of the source table.
//
// legacy.wp_usermeta.v1: the meta_key scope is unresolved. The WordPress
// usermeta table can hold personal and credential-adjacent data (for example
// session or capability entries) that Slice 4 does not need. Therefore:
//
//   - no future MySQL/source extractor may extract all wp_usermeta rows by
//     default;
//   - an explicit meta_key allowlist / Owner decision (D-23, TAD (E0) #18) is
//     required before any real extraction;
//   - importing an intentionally supplied (synthetic) snapshot in Slice 3 does
//     NOT resolve that decision.
//
// This register defines no allowlist. Any extractor must call
// ExtractionScopeUnresolved and refuse to extract a listed spec until the
// decision is recorded and this register is changed in a reviewed PR.
var unresolvedExtractionScope = map[string]string{
	"legacy.wp_usermeta.v1": "meta_key scope unresolved: no default full-table extraction; an Owner-approved meta_key allowlist (D-23, TAD (E0) #18) is required before real extraction",
}

// ExtractionScopeUnresolved reports whether real source extraction for the
// spec is blocked pending an Owner decision on its row scope, with the reason.
func ExtractionScopeUnresolved(specID string) (string, bool) {
	reason, ok := unresolvedExtractionScope[specID]
	return reason, ok
}
