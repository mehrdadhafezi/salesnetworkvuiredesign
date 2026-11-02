// Package biawin names the biawin module (Architecture V1 §13): the
// Subscription / SubscriptionItem representation of Biawin service delivery.
// Slice 10 implements only the READ / SHADOW subset: an offline import of
// normalized legacy evidence into insert-only, run-pinned shadow tables and
// a GET-only read surface. There is no activation, suspension, termination,
// order, wallet charge or any other command, and no provider call (D-35,
// D-38 unresolved: read-only representation, legacy remains master).
package biawin

import "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"

// Module is the biawin write owner: schema "biawin", NOLOGIN owner role
// "crm_owner_biawin". Only the offline shadow import writes it.
const Module owner.Module = "biawin"
