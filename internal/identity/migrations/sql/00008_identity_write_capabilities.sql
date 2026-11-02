-- Slice 11: controlled non-financial write capabilities (reference data
-- only). No role bundle receives them: there is no frozen role / capability
-- mapping for these commands, so they are granted to nobody (fail closed).
-- There is no invoice-issue, return, release, reset, transfer, allocation,
-- ready, HR final-apply, credential or grant capability.

-- +goose Up
INSERT INTO identity.capability (capability_key, description) VALUES
    ('cases.contact_outcome.record', 'Record the held lead''s contact outcome (no invoice, custody or finance effect)'),
    ('org.hr_request.create',        'Create an HR transfer / termination request for an in-scope seller (no personnel change)'),
    ('org.hr_request.review',        'Step-approve or reject an HR request as its current reviewer (no final apply)'),
    ('org.hr_request.read',          'Read HR requests the caller requested or currently reviews'),
    ('integration.writeback.read',   'Read legacy write-back diagnostics (counts, states, attempts; no payloads)');

-- +goose Down
DELETE FROM identity.capability WHERE capability_key IN ('cases.contact_outcome.record', 'org.hr_request.create', 'org.hr_request.review',
    'org.hr_request.read', 'integration.writeback.read');
