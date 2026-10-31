package salesbad

// Violates foreign-schema-write: the connector never writes cases directly.
const q = "INSERT INTO cases.work_case (case_id) VALUES ($1)"
