-- Slice 5: customer read capabilities (reference data only). No role
-- bundle receives them: customer-identity role assignment is not evidenced
-- (D-30 open), so they are granted to nobody.

-- +goose Up
INSERT INTO identity.capability (capability_key, description) VALUES
    ('customers.customer.read',        'Read shadow customers, their masked contact points, identifiers and evidence'),
    ('customers.merge_case.read',      'Read the merge-case candidate queue (no merge authority)'),
    ('customers.identity_report.read', 'Read the customer identity resolution report');

-- +goose Down
DELETE FROM identity.capability WHERE capability_key IN ('customers.customer.read', 'customers.merge_case.read', 'customers.identity_report.read');
