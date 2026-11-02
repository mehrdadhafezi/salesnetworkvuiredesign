// Package finance names the finance module (Architecture V1 §16). Slice 9
// implements only the SHADOW subset: a derived double-entry journal built
// offline from published billing / payments facts and normalized legacy
// wallet evidence, reconciled against legacy balances. It is NOT the
// financial system of record: nothing here is settled, bank-confirmed,
// Finance-approved or a posted production ledger, and no command posts,
// approves, settles, refunds, credits or debits anything.
package finance

import "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"

// Module is the finance write owner: schema "finance", NOLOGIN owner role
// "crm_owner_finance". Only the offline shadow rebuild writes it.
const Module owner.Module = "finance"
