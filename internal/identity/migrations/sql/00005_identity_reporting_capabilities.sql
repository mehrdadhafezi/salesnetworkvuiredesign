-- Slice 8: MIS / reporting read capabilities (reference data only). No role
-- bundle receives them: no frozen MIS / Invoice Reviewer row scope exists
-- and D-23/D-45 are open, so they are granted to nobody (fail closed).

-- +goose Up
INSERT INTO identity.capability (capability_key, description) VALUES
    ('reporting.metrics.read',        'Read metric contracts and aggregate report projections'),
    ('reporting.mis.read',            'Read MIS overview, source quality and reconciliation projections'),
    ('reporting.invoice_review.read', 'Read the Invoice Review query surface (audited; no export)');

-- +goose Down
DELETE FROM identity.capability WHERE capability_key IN ('reporting.metrics.read', 'reporting.mis.read', 'reporting.invoice_review.read');
