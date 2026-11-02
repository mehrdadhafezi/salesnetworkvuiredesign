// Package writeback is the integration-owned legacy write-back boundary of
// the Slice 11 controlled strangler. Legacy remains the master: a target
// command commits its own aggregate, history and outbox event; integration
// consumes the event (inbox-deduplicated), writes back to legacy through the
// Connector port, and records every attempt, reconciliation and parity
// observation in its own schema. It never writes another module's schema
// and never mutates target history to pretend a write did not happen.
//
// Rules (Cross-role §14; Architecture V1 §15.7 pattern):
//   - transport success is not business success: only an Applied outcome
//     the connector reports as SUCCESS marks a command APPLIED;
//   - a transport error or an UNKNOWN outcome leaves the command UNKNOWN; the
//     next step is an Observe (reconcile), never a blind Apply. UNKNOWN never
//     becomes FAILED automatically;
//   - before every Apply the command is durably moved to UNKNOWN (write-ahead
//     claim), so a crash between the legacy call and the record can never
//     cause a second blind Apply;
//   - a known refusal (proven non-commit) is REJECTED with a reconciliation
//     record; target history stays as it is;
//   - parity compares the target's expected effect of THIS command with the
//     legacy observation of THIS command's subject and intent.
//
// Only internal/integration/legacy/** knows legacy storage; this package
// sees neutral intents only.
package writeback

import (
	"context"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
)

// Command types delivered by Slice 11.
const (
	TypeContactOutcome = "cases.contact_outcome"
	TypeHRCreate       = "organization.hr_request_create"
	TypeHRDecide       = "organization.hr_request_decide"
)

// Intent is one target command as integration delivers it. Fields are
// legacy-neutral (target ids, mapped values); the legacy connector maps them.
type Intent struct {
	CommandID      string
	IdempotencyKey string
	CorrelationID  string
	CommandType    string
	SubjectKind    string
	SubjectID      string
	Fields         map[string]string
	// DependsOnLegacyRef is the legacy reference produced by the command
	// this one depends on (an HR decision depends on its request's create).
	DependsOnLegacyRef string
}

// Applied is the connector's answer to Apply.
type Applied struct {
	Outcome   result.Status
	Reasons   []string
	LegacyRef string
	Detail    string
}

// Observation classifies what legacy holds for an intent.
type Observation string

// Observations.
const (
	Committed Observation = "COMMITTED" // exactly this intent's effect is in legacy
	Absent    Observation = "ABSENT"    // proven not in legacy
	Ambiguous Observation = "AMBIGUOUS" // several candidates / duplicate / contradicting evidence
	Unseen    Observation = "UNKNOWN"   // legacy could not be read
)

// Observed is the connector's answer to Observe.
type Observed struct {
	Observation Observation
	// State is the observed legacy effect in the same neutral keys as the
	// target's expected effect (Expected).
	State     map[string]string
	LegacyRef string
	Detail    string
}

// Connector is the published write-back port. Apply must be idempotent for
// the same intent (the legacy request token / reconciliation key derives
// from CommandID); Observe never writes.
type Connector interface {
	Apply(ctx context.Context, in Intent) (Applied, error)
	Observe(ctx context.Context, in Intent) (Observed, error)
}

// Expected returns the effect the target expects in legacy for an intent,
// in the neutral keys Observe reports.
func Expected(in Intent) map[string]string {
	out := map[string]string{}
	keep := func(keys ...string) {
		for _, k := range keys {
			if v, ok := in.Fields[k]; ok {
				out[k] = v
			}
		}
	}
	switch in.CommandType {
	case TypeContactOutcome:
		keep("outcome", "reason", "callback_at")
	case TypeHRCreate:
		keep("request_type", "subject_user_id", "target_parent_user_id", "status", "current_reviewer_user_id")
	case TypeHRDecide:
		keep("to_status", "next_reviewer_user_id")
	}
	return out
}
