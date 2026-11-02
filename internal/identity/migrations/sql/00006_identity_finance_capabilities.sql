-- Slice 9: finance SHADOW ledger read capabilities (reference data only).
-- No role bundle receives them: the Finance role architecture (F06, OPD-08)
-- has no frozen authority for the shadow ledger, so they are granted to
-- nobody (fail closed). They are READ capabilities only; no finance write,
-- posting, approval, settlement or refund capability exists.

-- +goose Up
INSERT INTO identity.capability (capability_key, description) VALUES
    ('finance.shadow.read',         'Read finance SHADOW runs, accounts and derived balances (not the system of record)'),
    ('finance.ledger.read',         'Read the finance SHADOW journal entries and lines (not the system of record)'),
    ('finance.reconciliation.read', 'Read the legacy vs SHADOW ledger reconciliation (not the system of record)');

-- +goose Down
DELETE FROM identity.capability WHERE capability_key IN ('finance.shadow.read', 'finance.ledger.read', 'finance.reconciliation.read');
