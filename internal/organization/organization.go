// Package organization names the organization module (Architecture V1 §5,
// §6, §21): persons, employment, positions, levels, org units, reporting
// assignments (and, in later slices, compensation, delegation and HR
// requests). In Slice 4 it holds only a read/shadow model imported from
// legacy staging. It is never a permission engine: hierarchy is scope
// context, authority comes from identity grants evaluated by authorization.
package organization

import "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"

// Module is the organization write owner: schema "organization", NOLOGIN
// owner role "crm_owner_organization". No other module may write its schema.
const Module owner.Module = "organization"
