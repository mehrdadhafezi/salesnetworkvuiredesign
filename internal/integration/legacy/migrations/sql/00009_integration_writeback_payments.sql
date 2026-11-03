-- Slice 12: the payments module becomes a write-back source. A payments
-- write-back command is a manual payment attempt whose receipts are all
-- READY (payments.attempt_evidence_ready). Only the CHECK vocabularies are
-- widened; no row, history or attempt is changed.

-- +goose Up
ALTER TABLE integration.writeback_command DROP CONSTRAINT writeback_command_module;
ALTER TABLE integration.writeback_command ADD CONSTRAINT writeback_command_module
    CHECK (source_module IN ('cases', 'organization', 'payments'));
ALTER TABLE integration.writeback_command DROP CONSTRAINT writeback_command_subject;
ALTER TABLE integration.writeback_command ADD CONSTRAINT writeback_command_subject
    CHECK (subject_kind IN ('case', 'hr_request', 'payment_attempt'));
ALTER TABLE integration.writeback_reconciliation DROP CONSTRAINT writeback_reconciliation_owner;
ALTER TABLE integration.writeback_reconciliation ADD CONSTRAINT writeback_reconciliation_owner
    CHECK (domain_owner IN ('cases', 'organization', 'payments'));

-- +goose Down
ALTER TABLE integration.writeback_reconciliation DROP CONSTRAINT writeback_reconciliation_owner;
ALTER TABLE integration.writeback_reconciliation ADD CONSTRAINT writeback_reconciliation_owner
    CHECK (domain_owner IN ('cases', 'organization'));
ALTER TABLE integration.writeback_command DROP CONSTRAINT writeback_command_subject;
ALTER TABLE integration.writeback_command ADD CONSTRAINT writeback_command_subject
    CHECK (subject_kind IN ('case', 'hr_request'));
ALTER TABLE integration.writeback_command DROP CONSTRAINT writeback_command_module;
ALTER TABLE integration.writeback_command ADD CONSTRAINT writeback_command_module
    CHECK (source_module IN ('cases', 'organization'));
