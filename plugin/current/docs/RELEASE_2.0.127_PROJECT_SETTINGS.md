# 2.0.127 — Independent project settings save and visible results

Project-tab saves now use a dedicated admin-only, nonce-protected AJAX endpoint.
Only project configuration is sent, as one JSON field to avoid max_input_vars
truncation. Existing project validation and routing rules remain authoritative.
Unrelated Dot Flow/commission settings no longer block this save operation.

Dedicated save buttons at both ends of the Projects tab display persistent
success/error messages, working state, duplicate-submit protection and network
failure feedback. The main save handler also delegates to this endpoint when
Projects is active. Independent buttons validate only project fields, preventing
unrelated hidden-tab browser constraints from blocking them.

The supervisor placeholder remains enabled. Project transactions now check
START/COMMIT results. Inactivity settings and their routing effect run only after
project validation and commit, with project caches invalidated first.

Validation: JS syntax passed. Payload test with 600 routes, sparse indexes,
multi-select values, Unicode and escaped strings passed. PHP lint, WordPress
integration and browser validation are unavailable in this environment. Exact
production failure cannot be confirmed without the site's response/database.
