-- Slice 12: controlled financial write capabilities (reference data only).
-- No role bundle receives them: there is no frozen role / capability mapping
-- for these commands, so they are granted to nobody (fail closed). There is
-- no Finance approve / reject, posting, refund, wallet, settlement,
-- commission, invoice-issue, next-stage, return or gateway capability.

-- +goose Up
INSERT INTO identity.capability (capability_key, description) VALUES
    ('payments.upload.stage',    'Stage a receipt file in quarantine evidence storage (not evidence until bound)'),
    ('payments.batch.submit',    'Submit a manual payment batch for an owned invoice''s current stage (no review, posting or stage change)'),
    ('payments.submission.read', 'Read the payment submission context and own payment batches'),
    ('payments.evidence.read',   'Read payment evidence diagnostics and evidence reconciliation records'),
    ('finance.review.read',      'Read the Finance review queue and its decision gate (no approve / reject authority)');

-- +goose Down
DELETE FROM identity.capability WHERE capability_key IN ('payments.upload.stage', 'payments.batch.submit', 'payments.submission.read',
    'payments.evidence.read', 'finance.review.read');
