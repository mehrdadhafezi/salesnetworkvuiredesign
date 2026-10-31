// Package identity names the identity module (Architecture V1 §5, §6, §21):
// users, credentials, sessions, roles, capabilities and role grants. In
// Slice 4 it holds only a read/shadow model: user accounts imported from
// legacy staging, the capability vocabulary, role bundles and SHADOW role
// grants. Credentials and sessions are not operational and have no tables.
package identity

import "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"

// Module is the identity write owner: schema "identity", NOLOGIN owner role
// "crm_owner_identity". No other module may write its schema.
const Module owner.Module = "identity"
