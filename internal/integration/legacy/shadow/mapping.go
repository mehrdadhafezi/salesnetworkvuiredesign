package shadow

// Legacy → target value mappings. This is the only place that knows legacy
// role names, legacy status strings and the legacy HR access rule. Each
// mapping cites its 2.0.160 evidence; a value that is not listed is never
// guessed: it stays raw and maps to UNMAPPED / nil.

import (
	"regexp"

	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/refs"
	organizationapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/api"
)

// SourceSystem is the legacy system of every Slice 4 source (WordPress DB:
// core tables and the plugin's own tables).
const SourceSystem = string(refs.LegacyWP)

// legacyRoles maps 2.0.160 plugin WordPress roles to target roles.
// Evidence: includes/class-sn-activator.php:152-168 (role registration);
// Architecture V1 §4A (target roles) and §18 (the three finance roles
// collapse into one finance role with distinct capabilities). Roles not
// listed (converter, after-sales, shipping, operations, campaign partner,
// project roles, administrator) have no Slice 4 target role: UNMAPPED.
var legacyRoles = map[string]identityapi.Role{
	"sn_seller":             identityapi.RoleSeller,
	"sn_supervisor":         identityapi.RoleSupervisor,
	"sn_senior_supervisor":  identityapi.RoleSeniorSupervisor,
	"sn_sales_manager":      identityapi.RoleSalesManager,
	"sn_sales_deputy":       identityapi.RoleSalesDeputy,
	"sn_mis":                identityapi.RoleMIS,
	"sn_hr":                 identityapi.RoleHR,
	"sn_financial":          identityapi.RoleFinance,
	"sn_financial_approval": identityapi.RoleFinance,
	"sn_finance":            identityapi.RoleFinance,
	"sn_invoice_reviewer":   identityapi.RoleInvoiceReviewer,
}

// MapLegacyRole maps an exact raw legacy role label. Position slugs that
// some legacy write paths store in the same column (class-sn-plugin.php:11212)
// are NOT roles and stay unmapped: Position ≠ Role.
func MapLegacyRole(raw string) *identityapi.Role {
	if r, ok := legacyRoles[raw]; ok {
		return &r
	}
	return nil
}

// employmentStatuses are the statuses the 2.0.160 HR UI offers
// (class-sn-plugin.php:10135, 10155, 10945, 11675).
var employmentStatuses = map[string]organizationapi.EmploymentStatus{
	"active":     organizationapi.StatusActive,
	"inactive":   organizationapi.StatusInactive,
	"suspended":  organizationapi.StatusSuspended,
	"resigned":   organizationapi.StatusResigned,
	"terminated": organizationapi.StatusTerminated,
	"probation":  organizationapi.StatusProbation,
}

// MapEmploymentStatus maps a raw status; unknown values are UNMAPPED and a
// missing value is NOT_RECORDED (never defaulted to active).
func MapEmploymentStatus(raw *string) organizationapi.EmploymentStatus {
	if raw == nil || *raw == "" {
		return organizationapi.StatusNotRecorded
	}
	if s, ok := employmentStatuses[*raw]; ok {
		return s
	}
	return organizationapi.StatusUnmapped
}

// reportsTo is the only relationship type 2.0.160 writes and reads
// (class-sn-hierarchy-service.php:16, :55; class-sn-migration-service.php:585).
const reportsTo = "reports_to"

// MapRelationship maps a raw relationship type.
func MapRelationship(raw *string) organizationapi.RelationshipKind {
	if raw != nil && *raw == reportsTo {
		return organizationapi.ReportsTo
	}
	return organizationapi.RelationshipUnmapped
}

// hrPositionSlug is the HR position slug of the legacy HR access gate.
const hrPositionSlug = "hr"

// LegacyHRGate documents the 2.0.160 HR panel gate (class-sn-plugin.php:
// 9144-9163; class-sn-scope-service.php:23-27, 517-521; class-sn-hr-service.php:
// 85-91): manage_options OR WordPress role sn_hr OR the viewer's HR profile
// position slug "hr" (the profile is read with LIMIT 1 and no ORDER BY).
// Only the profile-position branch is provable from staged evidence; the
// role and manage_options branches need WordPress role assignments, which
// are not staged (wp_usermeta meta_value is never parsed; its scope is an
// open Owner decision). Every Slice 4 HR read capability maps to this one
// legacy gate.
const LegacyHRGate = "class-sn-plugin.php:9144-9163 (manage_options | role sn_hr | profile position hr)"

var (
	keyPattern = regexp.MustCompile(`^[1-9][0-9]{0,19}$`)
	intPattern = regexp.MustCompile(`^-?(0|[1-9][0-9]{0,8})$`)
)

// key reads a raw legacy reference: a canonical key, or nil when not
// recorded ("", "0", NULL) or unreadable (reported by the caller).
func key(raw *string) (k *string, unreadable bool) {
	if raw == nil || *raw == "" || *raw == "0" {
		return nil, false
	}
	if keyPattern.MatchString(*raw) {
		v := *raw
		return &v, false
	}
	return nil, true
}

// flag reads a raw TINYINT(1): "1" true, "0" false, anything else unknown.
func flag(raw *string) *bool {
	if raw == nil {
		return nil
	}
	switch *raw {
	case "1":
		t := true
		return &t
	case "0":
		f := false
		return &f
	}
	return nil
}

// integer reads a raw INT; anything not canonical is unknown.
func integer(raw *string) *int {
	if raw == nil || !intPattern.MatchString(*raw) {
		return nil
	}
	n := 0
	neg := false
	s := *raw
	if s[0] == '-' {
		neg, s = true, s[1:]
	}
	for _, c := range s {
		n = n*10 + int(c-'0')
	}
	if neg {
		n = -n
	}
	return &n
}

// text returns raw text, keeping NULL as nil (empty string stays empty).
func text(raw *string) *string { return raw }
