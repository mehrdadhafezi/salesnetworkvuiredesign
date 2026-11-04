-- Slice 13: commission controlled SHADOW read capabilities (reference data
-- only). No role bundle receives them (fail closed). There is no calculate,
-- approve, reject, post, credit, settle or reverse capability: approval and
-- posting are decision-blocked (OPD-09, D-41).

-- +goose Up
INSERT INTO identity.capability (capability_key, description) VALUES
    ('commission.shadow.read', 'Read commission SHADOW runs, stored calculations, parity, findings and legacy runs (no approve / post authority)'),
    ('commission.policy.read', 'Read observed commission policy versions (none approved)'),
    ('commission.own.read',    'Read the caller''s own historical commission credits (own scope only)');

-- +goose Down
DELETE FROM identity.capability WHERE capability_key IN ('commission.shadow.read', 'commission.policy.read', 'commission.own.read');
