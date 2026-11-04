-- Catalog service model (Architecture V1 §13; Slice 14). Written only by
-- the offline administrative recorder with an external Human Owner approval
-- reference; read by biawin and fulfillment through the catalog published
-- API. Every table is INSERT-only (grants + 00002 guards): a definition
-- change is a new version, an ownership change is a new scope event, and
-- history (who was responsible when) is never rewritten.
-- No PII column; no price, credit or money column.

CREATE TABLE catalog.service_version (
    service_key          text        NOT NULL,
    version              integer     NOT NULL,
    title                text        NOT NULL,
    workflow_key         text        NOT NULL,
    evidence_requirements jsonb      NOT NULL,
    completion_outcomes  text[]      NOT NULL,
    financial_behaviour  text        NOT NULL,
    integration_provider text        NOT NULL,
    status               text        NOT NULL,
    approval_ref         text        NOT NULL,
    recorded_by          text        NOT NULL,
    recorded_at          timestamptz NOT NULL,
    PRIMARY KEY (service_key, version),
    CONSTRAINT service_version_key CHECK (service_key ~ '^[a-z][a-z0-9_.-]{1,80}$'),
    CONSTRAINT service_version_positive CHECK (version > 0),
    CONSTRAINT service_version_workflow CHECK (workflow_key IN ('operational_task_v1')),
    CONSTRAINT service_version_evidence CHECK (jsonb_typeof(evidence_requirements) = 'array'),
    CONSTRAINT service_version_outcomes CHECK (cardinality(completion_outcomes) > 0
        AND completion_outcomes <@ ARRAY['EXECUTED', 'NOT_EXECUTED']::text[]),
    CONSTRAINT service_version_financial CHECK (financial_behaviour IN ('NONE', 'CONSUMES_CREDIT', 'CHARGES_LEDGER')),
    CONSTRAINT service_version_provider CHECK (integration_provider IN ('NONE', 'BIAWIN')),
    CONSTRAINT service_version_status CHECK (status IN ('ACTIVE', 'RETIRED')),
    CONSTRAINT service_version_approval CHECK (length(approval_ref) BETWEEN 3 AND 200),
    CONSTRAINT service_version_recorder CHECK (length(recorded_by) BETWEEN 1 AND 128)
);

-- The service components of a subscription product version (one product
-- may carry 1 … 30+ services; each becomes its own SubscriptionItem).
CREATE TABLE catalog.subscription_component (
    product_key      text        NOT NULL,
    product_version  integer     NOT NULL,
    seq              integer     NOT NULL,
    service_key      text        NOT NULL,
    service_version  integer     NOT NULL,
    approval_ref     text        NOT NULL,
    recorded_by      text        NOT NULL,
    recorded_at      timestamptz NOT NULL,
    PRIMARY KEY (product_key, product_version, seq),
    CONSTRAINT subscription_component_service FOREIGN KEY (service_key, service_version) REFERENCES catalog.service_version (service_key, version),
    CONSTRAINT subscription_component_seq CHECK (seq > 0 AND product_version > 0),
    CONSTRAINT subscription_component_approval CHECK (length(approval_ref) BETWEEN 3 AND 200)
);

-- Per-service responsibility scope events. A GRANT makes a user the
-- supervisor (or an executor) of one service; a REVOKE ends it. The current
-- scope is derived; past events are never rewritten, so a changed
-- supervisor never rewrites historical assignments or outcomes. A scope is
-- row-level authority for ONE service only, never a role or hierarchy tier.
CREATE TABLE catalog.service_scope_event (
    event_id      uuid        PRIMARY KEY,
    seq           bigint      GENERATED ALWAYS AS IDENTITY UNIQUE,
    service_key   text        NOT NULL,
    user_id       uuid        NOT NULL,
    scope_kind    text        NOT NULL,
    action        text        NOT NULL,
    approval_ref  text        NOT NULL,
    recorded_by   text        NOT NULL,
    recorded_at   timestamptz NOT NULL,
    CONSTRAINT service_scope_kind CHECK (scope_kind IN ('SUPERVISOR', 'EXECUTOR')),
    CONSTRAINT service_scope_action CHECK (action IN ('GRANT', 'REVOKE')),
    CONSTRAINT service_scope_approval CHECK (length(approval_ref) BETWEEN 3 AND 200)
);
CREATE INDEX service_scope_event_lookup ON catalog.service_scope_event (service_key, scope_kind, user_id, seq);

GRANT SELECT, INSERT ON catalog.service_version, catalog.subscription_component, catalog.service_scope_event TO crm_owner_catalog;
