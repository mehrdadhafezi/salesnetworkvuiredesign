-- Slice 10: Biawin / fulfillment READ shadow capabilities (reference data
-- only). No role bundle receives them: no frozen role authority exists for
-- the shadow subscription / fulfillment read, so they are granted to nobody
-- (fail closed). They are READ capabilities only; no activation, suspension,
-- termination, provider, shipping, assignment or completion capability
-- exists.

-- +goose Up
INSERT INTO identity.capability (capability_key, description) VALUES
    ('biawin.subscription.read', 'Read shadow Biawin subscriptions, their items, invoice links and history (no activation authority)'),
    ('fulfillment.case.read',    'Read shadow fulfillment cases, assignees, evidence metadata and history (no workflow authority)'),
    ('biawin.parity.read',       'Read the legacy vs target active subscription parity report');

-- +goose Down
DELETE FROM identity.capability WHERE capability_key IN ('biawin.subscription.read', 'fulfillment.case.read', 'biawin.parity.read');
