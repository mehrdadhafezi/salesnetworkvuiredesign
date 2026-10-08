# 2.0.133 — Combined pending financial review

Pending Review now includes the pending manual-deposit and uploaded-receipt
buckets plus remaining review cases. The dedicated tabs remain subset views.
The pending badge, summary card, list and amount share this combined scope.
All invoices still counts each invoice once, before aggregating the pending
badge. All = pending review + gateway + approved + rejected; do not add the two
pending subsets again. Updated panel copy and all-card caption explain this.

Validation: SQL fixtures passed for combined pending scope, full/partial approval,
rejection, resubmission, gateway payments and unique all count. JS syntax passed.
Live WordPress/MySQL verification was not available.
