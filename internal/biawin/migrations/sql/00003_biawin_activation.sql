-- Biawin activation contract (Architecture V1 §13; Slice 14). Target-owned,
-- separate from the run-pinned Slice 10 shadow facts.
--
-- One activation per business key (invoice + product version) records the
-- Biawin-owned decision on an InvoiceFullyPaid message: SHADOW_ACTIVATED,
-- BLOCKED (with reasons and the decisions blocking it) or NOT_APPLICABLE
-- (a Complaint or Archive branch never enters Biawin). D-35 is unresolved:
-- the only activation this schema can hold is a SHADOW one (CHECKs); a real
-- (production) activation, a Subscription+ membership or any credit grant
-- cannot be represented. Subscriptions and items are created only for a
-- SHADOW_ACTIVATED decision, one item per service component (1 … 30+), each
-- with its own identity. Everything is INSERT-only (append-only triggers).
-- No PII column, no money column.

-- +goose Up
CREATE TABLE biawin.activation (
    activation_id       uuid        PRIMARY KEY,
    business_key        text        NOT NULL UNIQUE,
    message_id          uuid        NOT NULL,
    correlation_id      text        NOT NULL,
    invoice_id          uuid        NOT NULL,
    invoice_ref         text        NULL,
    customer_link_state text        NOT NULL,
    customer_id         uuid        NULL,
    existing_customer   boolean     NOT NULL,
    customer_basis      text        NOT NULL,
    route               text        NOT NULL,
    route_basis         text        NOT NULL,
    product_key         text        NOT NULL,
    product_version     integer     NOT NULL,
    rollout_mode        text        NOT NULL,
    decision            text        NOT NULL,
    reasons             jsonb       NOT NULL,
    blocked_by          jsonb       NOT NULL,
    subscription_id     uuid        NULL,
    item_count          integer     NOT NULL,
    decided_at          timestamptz NOT NULL,
    CONSTRAINT activation_link CHECK (customer_link_state IN ('RESOLVED', 'UNKNOWN', 'CONFLICT')
        AND (customer_link_state = 'RESOLVED') = (customer_id IS NOT NULL)),
    CONSTRAINT activation_route CHECK (route IN ('ADDITIONAL_SERVICES', 'AFTER_SALES_COMPLAINT', 'ARCHIVE', 'UNKNOWN')),
    CONSTRAINT activation_mode CHECK (rollout_mode = 'SHADOW_COMPARE'),
    CONSTRAINT activation_decision CHECK (decision IN ('SHADOW_ACTIVATED', 'BLOCKED', 'NOT_APPLICABLE')),
    CONSTRAINT activation_subscription CHECK ((decision = 'SHADOW_ACTIVATED') = (subscription_id IS NOT NULL)),
    CONSTRAINT activation_branch CHECK (decision <> 'SHADOW_ACTIVATED' OR (route = 'ADDITIONAL_SERVICES' AND customer_link_state = 'RESOLVED')),
    CONSTRAINT activation_items CHECK (item_count >= 0 AND (decision = 'SHADOW_ACTIVATED' OR item_count = 0)),
    CONSTRAINT activation_arrays CHECK (jsonb_typeof(reasons) = 'array' AND jsonb_typeof(blocked_by) = 'array'),
    CONSTRAINT activation_correlation CHECK (length(correlation_id) BETWEEN 1 AND 128)
);

CREATE TABLE biawin.target_subscription (
    subscription_id  uuid        PRIMARY KEY,
    activation_id    uuid        NOT NULL UNIQUE REFERENCES biawin.activation (activation_id),
    invoice_id       uuid        NOT NULL,
    customer_id      uuid        NOT NULL,
    product_key      text        NOT NULL,
    product_version  integer     NOT NULL,
    state            text        NOT NULL,
    origin_mode      text        NOT NULL,
    created_at       timestamptz NOT NULL,
    CONSTRAINT target_subscription_state CHECK (state = 'SHADOW_ACTIVE'),
    CONSTRAINT target_subscription_origin CHECK (origin_mode = 'SHADOW_COMPARE')
);

CREATE TABLE biawin.target_item (
    item_id          uuid        PRIMARY KEY,
    subscription_id  uuid        NOT NULL REFERENCES biawin.target_subscription (subscription_id),
    seq              integer     NOT NULL,
    service_key      text        NOT NULL,
    service_version  integer     NOT NULL,
    state            text        NOT NULL,
    created_at       timestamptz NOT NULL,
    CONSTRAINT target_item_seq UNIQUE (subscription_id, seq),
    CONSTRAINT target_item_state CHECK (state = 'SHADOW_ACTIVE'),
    CONSTRAINT target_item_positive CHECK (seq > 0 AND service_version > 0)
);

CREATE TABLE biawin.activation_event (
    event_id       uuid        PRIMARY KEY,
    activation_id  uuid        NOT NULL REFERENCES biawin.activation (activation_id),
    kind           text        NOT NULL,
    detail         jsonb       NOT NULL,
    occurred_at    timestamptz NOT NULL,
    CONSTRAINT activation_event_kind CHECK (kind IN ('DECIDED', 'DUPLICATE_MESSAGE')),
    CONSTRAINT activation_event_detail CHECK (jsonb_typeof(detail) = 'object')
);

GRANT SELECT, INSERT ON biawin.activation, biawin.target_subscription, biawin.target_item, biawin.activation_event TO crm_owner_biawin;

CREATE TRIGGER activation_append_only BEFORE UPDATE OR DELETE ON biawin.activation FOR EACH ROW EXECUTE FUNCTION biawin.reject_change();
CREATE TRIGGER activation_no_truncate BEFORE TRUNCATE ON biawin.activation FOR EACH STATEMENT EXECUTE FUNCTION biawin.reject_change();
CREATE TRIGGER target_subscription_append_only BEFORE UPDATE OR DELETE ON biawin.target_subscription FOR EACH ROW EXECUTE FUNCTION biawin.reject_change();
CREATE TRIGGER target_subscription_no_truncate BEFORE TRUNCATE ON biawin.target_subscription FOR EACH STATEMENT EXECUTE FUNCTION biawin.reject_change();
CREATE TRIGGER target_item_append_only BEFORE UPDATE OR DELETE ON biawin.target_item FOR EACH ROW EXECUTE FUNCTION biawin.reject_change();
CREATE TRIGGER target_item_no_truncate BEFORE TRUNCATE ON biawin.target_item FOR EACH STATEMENT EXECUTE FUNCTION biawin.reject_change();
CREATE TRIGGER activation_event_append_only BEFORE UPDATE OR DELETE ON biawin.activation_event FOR EACH ROW EXECUTE FUNCTION biawin.reject_change();
CREATE TRIGGER activation_event_no_truncate BEFORE TRUNCATE ON biawin.activation_event FOR EACH STATEMENT EXECUTE FUNCTION biawin.reject_change();

-- +goose Down
DROP TABLE IF EXISTS biawin.activation_event;
DROP TABLE IF EXISTS biawin.target_item;
DROP TABLE IF EXISTS biawin.target_subscription;
DROP TABLE IF EXISTS biawin.activation;
