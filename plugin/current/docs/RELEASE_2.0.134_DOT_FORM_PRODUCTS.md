# 2.0.134 — Reserve Dot marketing form products

Products configured for the default Dot marketing form or named marketing forms
are excluded from shared seller-facing product selectors. Assigned products stay
reserved even while their form is disabled. Ordinary assessment products retain
the existing Dot enable/disable behavior. Manual invoice creation rejects these
form products server-side; controlled workflows retain their existing path.
The dedicated marketing-form invoice method is unchanged.

Validation: reviewed both selector call sites, manual invoice core guard and
marketing/subscription form invoice call paths. Live PHP/WordPress tests were
unavailable.

Gateway export investigated, not changed: it exports successful Zibal transaction
history independently of active finance-tab search/filter state. Local gateway
amount is multiplied by ten while CRM amount remains the invoice amount. Need
an example export or specific incorrect column/count to identify the reported
issue before changing its financial-report scope or amount semantics.
