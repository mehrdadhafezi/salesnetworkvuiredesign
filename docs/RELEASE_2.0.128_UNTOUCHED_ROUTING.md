# 2.0.128 — Apply saved Operations routes to untouched cases

Saving project Operations manager/supervisor routes now also refreshes existing
untouched Operations destinations for configured content products. Uses the
original source sales-manager snapshot, falling back to the membership snapshot.
Explicit source-manager/supervisor mappings take precedence over card defaults.

Eligibility: active membership, waiting project item, no manual assignment,
contact, follow-up, note, message, project action, upgrade invoice, payment state,
customer decision, completion or cancellation. Only creation/automatic-routing
and previous settings-reroute events are allowed; unknown historical events
prevent movement. Automatically assigned experts require a system routing event.

Existing customer-waiting cases retain their stage and decision deadline; queued
cases move to the new supervisor or manager queue. Actual routing stages do not
advance prematurely. Unchanged destinations retain automatic assignments.

Updates and old/new destination audit records occur inside the project-settings
transaction. Failures trigger rollback. Candidate rows are locked and rechecked
on update, scanned in batches of 200 without a total-row cap. The project save
message reports how many existing destinations were refreshed.

Validation: SQLite guard fixtures passed for untouched cases, history exclusions,
automatic vs manual assignment, repeated settings updates and waiting customers.
No live WordPress/MySQL or PHP runtime was available; transaction/ownership
integration still requires verification on the site.
