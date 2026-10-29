-- TEST-ONLY synthetic module "syn_beta": a consumer with one projection table.
CREATE TABLE syn_beta.widget_seen (
    event_id  uuid PRIMARY KEY,
    label     text NOT NULL CHECK (length(label) BETWEEN 1 AND 64),
    seen      integer NOT NULL DEFAULT 1
);
REVOKE ALL ON syn_beta.widget_seen FROM PUBLIC;
GRANT SELECT, INSERT, UPDATE ON syn_beta.widget_seen TO crm_owner_syn_beta;
