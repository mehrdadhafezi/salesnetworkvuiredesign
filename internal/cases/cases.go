// Package cases names the cases module (Architecture V1 §5, §10): the sales
// work unit (Case), its legacy Lead facts, source lineage (CaseSource),
// proof-backed links between representations (CaseAlias), append-only state
// history and immutable AttributionSnapshots. In Slice 6 it is a read/shadow
// model only: no CreateCase, RecordContactOutcome, MarkReady or any other
// command exists; cases are materialised solely by the controlled shadow
// import (one per staged legacy source row). OPD-01 stays open: links
// between representations are recorded, never merged.
package cases

import "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"

// Module is the cases write owner: schema "cases", NOLOGIN owner role
// "crm_owner_cases". No other module may write its schema.
const Module owner.Module = "cases"
