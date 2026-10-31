-- Slice 6: sales read capabilities (reference data only). No role bundle
-- receives them: OPD-01 is open and no frozen seller/supervisor/manager
-- row-scope policy exists, so they are granted to nobody (fail closed).

-- +goose Up
INSERT INTO identity.capability (capability_key, description) VALUES
    ('cases.case.read',      'Read shadow cases, their lineage, state history and attribution snapshots'),
    ('custody.custody.read', 'Read derived custody projections, custody events and findings'),
    ('sales.report.read',    'Read the sales source coverage and custody parity reports');

-- +goose Down
DELETE FROM identity.capability WHERE capability_key IN ('cases.case.read', 'custody.custody.read', 'sales.report.read');
