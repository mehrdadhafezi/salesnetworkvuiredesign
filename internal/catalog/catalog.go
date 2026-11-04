// Package catalog names the catalog module (Architecture V1 §13): the
// versioned ServiceDefinition / ServiceVersion (workflow version, evidence
// requirements, approved completion outcomes, financial behaviour
// classification, integration provider), the product → service components
// of a subscription product and the per-service responsibility scopes
// (supervisor / executor). Slice 14 writes it only through the offline
// administrative recorder (crm-catalog-admin), which requires an external
// Human Owner approval reference for every record: catalog never invents a
// service, a workflow, an outcome or an owner. Every table is append-only;
// a change is a new version or a new scope event, never an overwrite.
package catalog

import "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"

// Module is the catalog write owner: schema "catalog", NOLOGIN owner role
// "crm_owner_catalog".
const Module owner.Module = "catalog"
