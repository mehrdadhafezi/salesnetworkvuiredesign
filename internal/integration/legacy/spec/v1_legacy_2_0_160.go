package spec

// Published v1 extract specs for the official legacy baseline 2.0.160.
//
// Every column below was verified in plugin/current (Sales Network 2.0.160,
// SN_BUILD 2026-10-10-finance-dialog-layout). No column is taken from memory,
// later development releases (2.0.182 is behavioural overlay only), the target
// architecture or WordPress conventions. Columns that 2.0.160 adds at runtime
// (add_column_if_missing / ALTER TABLE) are included because 2.0.160 code
// ensures them; columns that only older installs may carry (for example the
// legacy title/name/position_key candidates read by backfill_hr_lookup_labels)
// are NOT verified as present and are therefore not extracted.
//
// These specs are immutable. Any change requires a new version.

const (
	migFile    = "plugin/current/includes/class-sn-migration-service.php"
	pluginFile = "plugin/current/includes/class-sn-plugin.php"
	transfer   = "plugin/current/includes/class-sn-hr-transfer.php"
)

const (
	rowOrderKeyAsc = "ascending by the single key column, compared as an unsigned decimal; strictly increasing (also proves uniqueness)"
	wpCoreType     = "UNKNOWN (WordPress core table; DDL not in plugin/current)"
)

func col(name, rawType string, n Nullability, c Classification, evidence string) Column {
	return Column{Name: name, RawType: rawType, Nullability: n, Classification: c, Evidence: evidence}
}

func colNote(name, rawType string, n Nullability, c Classification, evidence, note string) Column {
	cc := col(name, rawType, n, c, evidence)
	cc.Note = note
	return cc
}

func v1(entity, table string, cols []Column, ev []Evidence, notes ...string) Spec {
	return Spec{
		ID:                "legacy." + entity + ".v1",
		Entity:            entity,
		Version:           1,
		LegacyApplication: LegacyApplication,
		LegacyVersion:     OfficialLegacyVersion,
		SourceTable:       table,
		Columns:           cols,
		KeyColumns:        []string{cols[0].Name},
		KeyKind:           KeyUnsignedDecimal,
		RowOrder:          rowOrderKeyAsc,
		Checksum:          ChecksumStrategy,
		StagingTable:      "stg_" + entity + "_v1",
		Evidence:          ev,
		Notes:             append([]string{}, notes...),
	}
}

// V1 returns every published v1 spec for legacy 2.0.160: the Slice 3 set
// and the Slice 5 extension (v1_customer_2_0_160.go).
func V1() []Spec { return append(append(V1Slice3(), V1Slice5()...), V1Slice6()...) }

// Default returns the registry of every published spec. It panics only if a
// published spec is invalid, which the spec tests rule out.
func Default() *Registry {
	r, err := NewRegistry(V1()...)
	if err != nil {
		panic(err)
	}
	return r
}

// --- WordPress core tables as used by the 2.0.160 plugin -------------------

func wpUserV1() Spec {
	ev := transfer + ":263; " + pluginFile + ":9563 (SELECT u.user_login, u.user_email ... JOIN users)"
	return v1("wp_user", "users", []Column{
		col("ID", wpCoreType, NullUnknown, ClassNone, transfer+":263; "+pluginFile+":9563 (u.ID joins)"),
		colNote("user_login", wpCoreType, NullUnknown, ClassPersonal, ev, "may hold a phone number for customer accounts; preserved raw, never an identity key"),
		col("user_email", wpCoreType, NullUnknown, ClassPersonal, ev),
		col("display_name", wpCoreType, NullUnknown, ClassPersonal, pluginFile+":9563; "+pluginFile+":1044 (u.display_name)"),
		col("user_registered", wpCoreType, NullUnknown, ClassNone, pluginFile+":29995, :35518 ($user->user_registered)"),
	}, []Evidence{
		{File: transfer, Lines: "263", Note: "SELECT ... u.user_login, u.user_email ... FROM users"},
		{File: pluginFile, Lines: "9563, 29995, 35518", Note: "u.ID, u.display_name, u.user_login, u.user_email, user_registered"},
	},
		"WordPress core table; only the columns the 2.0.160 plugin references are extracted",
		"user_pass and every other credential/core column are deliberately NOT extracted",
	)
}

// wpUsermetaV1 defines the raw columns only. Its meta_key ROW SCOPE is an
// unresolved Owner decision: no future extractor may extract all usermeta rows
// by default; see ExtractionScopeUnresolved (scope.go). Slice 3 snapshot
// import does not resolve that decision.
func wpUsermetaV1() Spec {
	ev := transfer + ":402-404 (umeta_id, user_id, meta_key, meta_value)"
	return v1("wp_usermeta", "usermeta", []Column{
		col("umeta_id", wpCoreType, NullUnknown, ClassNone, ev),
		col("user_id", wpCoreType, NullUnknown, ClassNone, ev),
		col("meta_key", wpCoreType, NullUnknown, ClassNone, ev),
		colNote("meta_value", wpCoreType, NullUnknown, ClassSensitive, ev,
			"raw serialized/meta text preserved exactly; may contain names, phones, role maps or other unverified content; never parsed in staging"),
	}, []Evidence{
		{File: transfer, Lines: "402-404", Note: "MAX(umeta_id) ... WHERE user_id ... AND meta_key=..., meta_value"},
		{File: pluginFile, Lines: "1512, 12955, 17718", Note: "user_id / meta_key / meta_value reads"},
	},
		"WordPress core table; row scope and any meta_key exclusion are an owner decision (D-23 / TAD (E0) #18); v1 defines no filter",
	)
}

// --- HR / organisation tables -------------------------------------------------

func lookupCols(createLines map[string]string, addLines map[string]string, panelKey bool) []Column {
	created := func(name string) string { return migFile + ":" + createLines[name] + " (table definition)" }
	both := func(name string) string {
		return migFile + ":" + createLines[name] + " (table definition); :" + addLines[name] + " (add_column_if_missing)"
	}
	cols := []Column{
		col("id", "BIGINT UNSIGNED", NotNull, ClassNone, created("id")),
		col("slug", "VARCHAR(80)", NotNull, ClassNone, both("slug")),
		col("label", "VARCHAR(120)", NotNull, ClassNone, both("label")),
		col("description", "TEXT", Nullable, ClassNone, both("description")),
	}
	if panelKey {
		cols = append(cols, col("panel_key", "VARCHAR(80)", Nullable, ClassNone, both("panel_key")))
	}
	return append(cols,
		col("sort_order", "INT", NotNull, ClassNone, both("sort_order")),
		col("is_active", "TINYINT(1)", NotNull, ClassNone, both("is_active")),
		col("is_system", "TINYINT(1)", NotNull, ClassNone, both("is_system")),
		colNote("created_at", "DATETIME", NullUnknown, ClassNone, both("created_at"),
			"NOT NULL in CREATE, NULL when added later; raw legacy DATETIME, zone not asserted"),
		colNote("updated_at", "DATETIME", NullUnknown, ClassNone, both("updated_at"),
			"NOT NULL in CREATE, NULL when added later; raw legacy DATETIME, zone not asserted"),
	)
}

func hrPositionV1() Spec {
	return v1("hr_position", "sn_hr_positions", lookupCols(
		map[string]string{"id": "389", "slug": "390", "label": "391", "description": "392", "panel_key": "393", "sort_order": "394", "is_active": "395", "is_system": "396", "created_at": "397", "updated_at": "398"},
		map[string]string{"slug": "336", "label": "337", "description": "338", "panel_key": "339", "sort_order": "340", "is_active": "341", "is_system": "342", "created_at": "343", "updated_at": "344"},
		true,
	), []Evidence{{File: migFile, Lines: "388-403, 335-346", Note: "create_hr_tables + ensure_hr_lookup_schema"}},
		"legacy columns that older installs may carry (title, name, position_key) are not verified in 2.0.160 and are not extracted")
}

func hrLevelV1() Spec {
	return v1("hr_level", "sn_hr_levels", lookupCols(
		map[string]string{"id": "406", "slug": "407", "label": "408", "description": "409", "sort_order": "410", "is_active": "411", "is_system": "412", "created_at": "413", "updated_at": "414"},
		map[string]string{"slug": "349", "label": "350", "description": "351", "sort_order": "352", "is_active": "353", "is_system": "354", "created_at": "355", "updated_at": "356"},
		false,
	), []Evidence{{File: migFile, Lines: "405-419, 348-358", Note: "create_hr_tables + ensure_hr_lookup_schema"}},
		"legacy columns that older installs may carry (title, name, level_key) are not verified in 2.0.160 and are not extracted")
}

func hrProfileV1() Spec {
	c := func(line string) string { return migFile + ":" + line + " (table definition)" }
	a := func(line string) string { return migFile + ":" + line + " (add_column_if_missing)" }
	return v1("hr_profile", "sn_hr_profiles", []Column{
		col("id", "BIGINT UNSIGNED", NotNull, ClassNone, c("424")),
		col("user_id", "BIGINT UNSIGNED", NotNull, ClassNone, c("425")),
		col("hr_display_name", "VARCHAR(191)", Nullable, ClassPersonal, pluginFile+":1043, :8892 (ADD COLUMN statement)"),
		col("position_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("426")+"; "+a("479")),
		col("level_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("427")+"; "+a("480")),
		col("employee_code", "VARCHAR(100)", Nullable, ClassPersonal, a("481")),
		col("employment_status", "VARCHAR(50)", NotNull, ClassPersonal, c("428")+"; "+a("482")),
		col("employment_type", "VARCHAR(50)", Nullable, ClassPersonal, a("483")),
		col("contract_status", "VARCHAR(50)", Nullable, ClassPersonal, c("429")),
		col("training_status", "VARCHAR(50)", Nullable, ClassPersonal, c("430")),
		col("hire_date", "DATE", Nullable, ClassPersonal, a("484")),
		col("termination_date", "DATE", Nullable, ClassPersonal, a("485")),
		colNote("base_salary", "BIGINT UNSIGNED", Nullable, ClassSensitive, a("486"), "raw value; no unit interpretation or conversion in staging"),
		colNote("salary_currency", "VARCHAR(20)", NotNull, ClassNone, a("487"), "raw legacy unit label; not interpreted"),
		col("national_id", "VARCHAR(50)", Nullable, ClassSensitive, a("488")),
		colNote("work_phone", "VARCHAR(50)", Nullable, ClassPersonal, a("489"), "raw; never normalised or used as identity"),
		colNote("emergency_phone", "VARCHAR(50)", Nullable, ClassPersonal, a("490"), "raw; never normalised or used as identity"),
		col("department_id", "BIGINT UNSIGNED", Nullable, ClassNone, a("491")),
		col("unit_id", "BIGINT UNSIGNED", Nullable, ClassNone, a("492")),
		col("team_id", "BIGINT UNSIGNED", Nullable, ClassNone, a("493")),
		colNote("notes", "TEXT", Nullable, ClassSensitive, a("494"), "free text about a person; content unverified"),
		col("legacy_role", "VARCHAR(120)", Nullable, ClassNone, c("431")),
		col("legacy_source", "VARCHAR(120)", Nullable, ClassNone, c("432")),
		col("is_active", "TINYINT(1)", NotNull, ClassNone, c("433")),
		col("created_at", "DATETIME", NotNull, ClassNone, c("434")),
		col("updated_at", "DATETIME", NotNull, ClassNone, c("435")),
	}, []Evidence{
		{File: migFile, Lines: "423-442", Note: "table definition of sn_hr_profiles"},
		{File: migFile, Lines: "477-497", Note: "add_column_if_missing for HR profile columns"},
		{File: pluginFile, Lines: "1039-1044, 8885-8893", Note: "ADD COLUMN hr_display_name"},
	})
}

func hrPositionRoleMappingV1() Spec {
	c := func(line string) string { return migFile + ":" + line + " (table definition)" }
	return v1("hr_position_role_mapping", "sn_hr_position_role_mappings", []Column{
		col("id", "BIGINT UNSIGNED", NotNull, ClassNone, c("445")),
		colNote("legacy_role", "VARCHAR(120)", NotNull, ClassNone, c("446"), "raw legacy role string; never mapped to a capability in staging"),
		col("position_id", "BIGINT UNSIGNED", NotNull, ClassNone, c("447")),
		col("default_level_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("448")),
		col("is_active", "TINYINT(1)", NotNull, ClassNone, c("449")),
		col("created_at", "DATETIME", NotNull, ClassNone, c("450")),
		col("updated_at", "DATETIME", NotNull, ClassNone, c("451")),
	}, []Evidence{{File: migFile, Lines: "444-457", Note: "table definition of sn_hr_position_role_mappings"}})
}

func structureCols(parent string) []Column {
	c := func(line string) string { return migFile + ":" + line + " (table definition)" }
	both := func(cl, al string) string { return c(cl) + "; " + migFile + ":" + al + " (add_column_if_missing)" }
	cols := []Column{col("id", "BIGINT UNSIGNED", NotNull, ClassNone, c("535"))}
	if parent != "" {
		cols = append(cols, col(parent, "BIGINT UNSIGNED", Nullable, ClassNone, migFile+":525/529 (parent_column), :551 (add_column_if_missing)"))
	}
	return append(cols,
		col("slug", "VARCHAR(100)", NotNull, ClassNone, both("537", "555")),
		col("label", "VARCHAR(191)", NotNull, ClassNone, both("538", "556")),
		col("description", "TEXT", Nullable, ClassNone, both("539", "557")),
		col("is_active", "TINYINT(1)", NotNull, ClassNone, both("540", "558")),
		col("sort_order", "INT", NotNull, ClassNone, both("541", "559")),
		col("created_at", "DATETIME", Nullable, ClassNone, both("542", "560")),
		col("updated_at", "DATETIME", Nullable, ClassNone, both("543", "561")),
	)
}

func structureEvidence(table string) []Evidence {
	return []Evidence{{File: migFile, Lines: "515-575", Note: "create_hr_structure_tables: " + table}}
}

func hrDepartmentV1() Spec {
	return v1("hr_department", "sn_hr_departments", structureCols(""), structureEvidence("sn_hr_departments (no parent column)"))
}

func hrUnitV1() Spec {
	return v1("hr_unit", "sn_hr_units", structureCols("department_id"), structureEvidence("sn_hr_units (parent department_id)"))
}

func hrTeamV1() Spec {
	return v1("hr_team", "sn_hr_teams", structureCols("unit_id"), structureEvidence("sn_hr_teams (parent unit_id)"))
}

func hrAssignmentV1() Spec {
	c := func(line string) string { return migFile + ":" + line + " (table definition)" }
	return v1("hr_assignment", "sn_hr_assignments", []Column{
		col("id", "BIGINT UNSIGNED", NotNull, ClassNone, c("582")),
		col("child_profile_id", "BIGINT UNSIGNED", NotNull, ClassNone, c("583")),
		col("parent_profile_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("584")),
		colNote("relationship_type", "VARCHAR(80)", NotNull, ClassNone, c("585"), "raw; unfamiliar values are preserved"),
		colNote("effective_from", "DATETIME", Nullable, ClassNone, c("586"), "raw legacy DATETIME; zone not asserted"),
		colNote("effective_to", "DATETIME", Nullable, ClassNone, c("587"), "raw legacy DATETIME; zone not asserted"),
		col("is_current", "TINYINT(1)", NotNull, ClassNone, c("588")),
		col("source", "VARCHAR(80)", Nullable, ClassNone, c("589")),
		col("created_by", "BIGINT UNSIGNED", Nullable, ClassNone, c("590")),
		col("created_at", "DATETIME", NotNull, ClassNone, c("591")),
		col("updated_at", "DATETIME", NotNull, ClassNone, c("592")),
	}, []Evidence{{File: migFile, Lines: "581-600", Note: "table definition of sn_hr_assignments"}})
}

func hrAssignmentHistoryV1() Spec {
	c := func(line string) string { return migFile + ":" + line + " (table definition)" }
	return v1("hr_assignment_history", "sn_hr_assignment_history", []Column{
		col("id", "BIGINT UNSIGNED", NotNull, ClassNone, c("603")),
		col("assignment_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("604")),
		col("child_profile_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("605")),
		col("old_parent_profile_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("606")),
		col("new_parent_profile_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("607")),
		col("actor_user_id", "BIGINT UNSIGNED", Nullable, ClassNone, c("608")),
		col("action", "VARCHAR(120)", NotNull, ClassNone, c("609")),
		colNote("reason", "TEXT", Nullable, ClassPersonal, c("610"), "free text; content unverified"),
		colNote("context", "LONGTEXT", Nullable, ClassPersonal, c("611"), "raw JSON/text preserved exactly; never parsed in staging"),
		col("created_at", "DATETIME", NotNull, ClassNone, c("612")),
	}, []Evidence{{File: migFile, Lines: "602-621", Note: "table definition of sn_hr_assignment_history"}})
}
