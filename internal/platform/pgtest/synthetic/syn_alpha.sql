-- TEST-ONLY synthetic module "syn_alpha": a producer with one aggregate table.
CREATE TABLE syn_alpha.widget (
    widget_id  bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    label      text   NOT NULL CHECK (length(label) BETWEEN 1 AND 64)
);
REVOKE ALL ON syn_alpha.widget FROM PUBLIC;
GRANT SELECT, INSERT ON syn_alpha.widget TO crm_owner_syn_alpha;
