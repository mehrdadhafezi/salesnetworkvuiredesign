// Package customers names the customers module (Architecture V1 §5, §10):
// canonical customer subjects, contact points, customer identifiers,
// identity evidence and merge-case candidates. In Slice 5 it is a
// read/shadow model only: customers are minted solely by the controlled
// shadow import (one per legacy issued-to record), no merge is ever
// executed, and phone is never identity (D-29, D-30, D-31 stay open).
package customers

import "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"

// Module is the customers write owner: schema "customers", NOLOGIN owner
// role "crm_owner_customers". No other module may write its schema.
const Module owner.Module = "customers"
