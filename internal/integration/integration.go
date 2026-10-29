// Package integration names the integration module (Architecture V1 §5,
// §21, §24). It owns connectors, raw legacy staging, LegacyReference and
// sync cursors. Only its legacy sub-tree knows legacy table names; this
// package deliberately holds none.
package integration

import "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"

// Module is the integration write owner: schema "integration", NOLOGIN owner
// role "crm_owner_integration". No other module may write its schema.
const Module owner.Module = "integration"
