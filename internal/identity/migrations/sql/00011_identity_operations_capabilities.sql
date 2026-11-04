-- Slice 14: controlled fulfillment / Biawin capabilities (reference data
-- only). No role bundle receives them (fail closed); each is also row-scoped
-- per service by the catalog. There is no activation, wallet-charge,
-- settlement, refund, cancel, reopen or commission capability.

-- +goose Up
INSERT INTO identity.capability (capability_key, description) VALUES
    ('fulfillment.work.read',    'Read work cases of services the caller currently supervises or executes'),
    ('fulfillment.work.assign',  'Assign a work case of a service the caller currently supervises to a current executor of that service'),
    ('fulfillment.work.execute', 'Record actions, evidence metadata and the outcome of a work case assigned to the caller'),
    ('biawin.activation.read',   'Read SHADOW Biawin activation decisions and their reconciliation context'),
    ('integration.provider_operation.read',      'Read external provider operations and their reconciliation state'),
    ('integration.provider_operation.reconcile', 'Record human reconciliation evidence for an UNKNOWN provider operation (never dispatches)');

-- +goose Down
DELETE FROM identity.capability WHERE capability_key IN ('fulfillment.work.read', 'fulfillment.work.assign', 'fulfillment.work.execute',
    'biawin.activation.read', 'integration.provider_operation.read', 'integration.provider_operation.reconcile');
