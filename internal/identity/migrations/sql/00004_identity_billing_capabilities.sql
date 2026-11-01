-- Slice 7: billing + payments read capabilities (reference data only). No
-- role bundle receives them: no frozen Finance / seller / supervisor row
-- scope exists (D-25, D-26, D-28 open), so they are granted to nobody
-- (fail closed).

-- +goose Up
INSERT INTO identity.capability (capability_key, description) VALUES
    ('billing.invoice.read',        'Read shadow invoices, lines, payment stages, state and total evidence'),
    ('payments.attempt.read',       'Read shadow payment attempts, gateway evidence (hashed) and receipt reference metadata'),
    ('billing.reconciliation.read', 'Read the billing and payments coverage report');

-- +goose Down
DELETE FROM identity.capability WHERE capability_key IN ('billing.invoice.read', 'payments.attempt.read', 'billing.reconciliation.read');
