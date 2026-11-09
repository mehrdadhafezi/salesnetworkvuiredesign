// Package posting is the R4 ledger posting CONTRACT (Architecture V1 §16).
// It defines and validates a posting intent and proves its invariants on a
// SYNTHETIC journal. It has no production store, no route, no command and
// no posting authority: D-41 (accrual vs cash), OPD-06 (units, rounding),
// OPD-08 (approval) and OPD-09 (entitlement routing) are open, so
// EvaluateActivation always answers BLOCKED.
//
// Invariants:
//   - a stable financial business key identifies one economic effect; the
//     same key with the same request is a replay (no new effect), with a
//     different request a CONFLICT;
//   - one explicit unit per intent; amounts are positive int64 minor units;
//     no float, no conversion, no rounding;
//   - DEBITS == CREDITS (overflow-checked);
//   - provenance: the source system and at least one source invoice or
//     payment reference;
//   - recorded time and effective time are distinct, explicit, UTC;
//   - a reversal is a SEPARATE compensating intent that mirrors the
//     original exactly; the original is never mutated or deleted, and is
//     reversed at most once;
//   - an UNKNOWN store outcome is reconciled by key before any retry.
package posting

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"time"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/money"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
)

// Sides.
const (
	Debit  = "DEBIT"
	Credit = "CREDIT"
)

// Basis labels. D-41 is OPEN: neither is chosen; an intent must name one
// explicitly and activation is BLOCKED for both.
const (
	BasisAccrual = "ACCRUAL"
	BasisCash    = "CASH"
)

// Decision ids and engineering gaps blocking activation.
const (
	DecisionD41        = "D-41"
	DecisionOPD06      = "OPD-06"
	DecisionOPD08      = "OPD-08"
	DecisionOPD09      = "OPD-09"
	NoPostingAuthority = "NO_POSTING_AUTHORITY"
)

// Errors.
var (
	ErrInvalid    = errors.New("posting: invalid intent")
	ErrUnbalanced = errors.New("posting: debits != credits")
	ErrOverflow   = errors.New("posting: amount overflow")
	// ErrOutcomeUnknown is returned by a store when a commit's outcome
	// cannot be determined; the key must be reconciled before any retry.
	ErrOutcomeUnknown = errors.New("posting: store outcome unknown")
)

var (
	keyPattern     = regexp.MustCompile(`^[a-z][a-z0-9_]*(\|[A-Za-z0-9_.:-]+){3,8}$`)
	accountPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{1,63}$`)
	tokenPattern   = regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,128}$`)
)

// Line is one ledger line.
type Line struct {
	Account string
	Side    string
	Amount  int64
}

// Source is the provenance of an intent.
type Source struct {
	System           string
	InvoiceID        *string
	PaymentAttemptID *string
}

// Intent is one proposed ledger transaction.
type Intent struct {
	BusinessKey string
	Basis       string
	Unit        money.Unit
	Lines       []Line
	Source      Source
	EffectiveAt time.Time
	RecordedAt  time.Time
	// Reverses is the business key of the intent this one compensates.
	Reverses *string
}

func utc(t time.Time) bool { return !t.IsZero() && t.Location() == time.UTC }

// Validate checks the structural invariants.
func (i Intent) Validate() error {
	switch {
	case !keyPattern.MatchString(i.BusinessKey):
		return fmt.Errorf("%w: business key %q", ErrInvalid, i.BusinessKey)
	case i.Basis != BasisAccrual && i.Basis != BasisCash:
		return fmt.Errorf("%w: basis must be named explicitly", ErrInvalid)
	case !i.Unit.Valid():
		return fmt.Errorf("%w: unit must be explicit", ErrInvalid)
	case !tokenPattern.MatchString(i.Source.System):
		return fmt.Errorf("%w: source system", ErrInvalid)
	case i.Source.InvoiceID == nil && i.Source.PaymentAttemptID == nil:
		return fmt.Errorf("%w: a source invoice or payment reference is required", ErrInvalid)
	case !utc(i.EffectiveAt) || !utc(i.RecordedAt):
		return fmt.Errorf("%w: effective and recorded times must be explicit UTC", ErrInvalid)
	case len(i.Lines) < 2:
		return fmt.Errorf("%w: at least two lines", ErrInvalid)
	}
	for _, p := range []*string{i.Source.InvoiceID, i.Source.PaymentAttemptID, i.Reverses} {
		if p != nil && !tokenPattern.MatchString(*p) && !keyPattern.MatchString(*p) {
			return fmt.Errorf("%w: reference %q", ErrInvalid, *p)
		}
	}
	var debit, credit int64
	for _, l := range i.Lines {
		if !accountPattern.MatchString(l.Account) || l.Amount <= 0 {
			return fmt.Errorf("%w: line %+v", ErrInvalid, l)
		}
		switch l.Side {
		case Debit:
			if debit > math.MaxInt64-l.Amount {
				return ErrOverflow
			}
			debit += l.Amount
		case Credit:
			if credit > math.MaxInt64-l.Amount {
				return ErrOverflow
			}
			credit += l.Amount
		default:
			return fmt.Errorf("%w: side %q", ErrInvalid, l.Side)
		}
	}
	if debit != credit {
		return ErrUnbalanced
	}
	return nil
}

// RequestHash is the canonical hash of an intent (lines sorted).
func RequestHash(i Intent) string {
	lines := append([]Line(nil), i.Lines...)
	sort.Slice(lines, func(a, b int) bool {
		if lines[a].Account != lines[b].Account {
			return lines[a].Account < lines[b].Account
		}
		if lines[a].Side != lines[b].Side {
			return lines[a].Side < lines[b].Side
		}
		return lines[a].Amount < lines[b].Amount
	})
	b, _ := json.Marshal(struct {
		Key, Basis, Unit string
		Lines            []Line
		Source           Source
		Effective, Rec   string
		Reverses         *string
	}{i.BusinessKey, i.Basis, i.Unit.String(), lines, i.Source, i.EffectiveAt.Format(time.RFC3339Nano), i.RecordedAt.Format(time.RFC3339Nano), i.Reverses})
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// Reversal builds the separate compensating intent of orig: mirrored
// lines, same unit and basis, same source, its own key and times.
func Reversal(orig Intent, key string, effective, recorded time.Time) (Intent, error) {
	if err := orig.Validate(); err != nil {
		return Intent{}, err
	}
	if orig.Reverses != nil {
		return Intent{}, fmt.Errorf("%w: a reversal is not reversed again; post a new intent", ErrInvalid)
	}
	r := orig
	r.BusinessKey, r.EffectiveAt, r.RecordedAt = key, effective, recorded
	k := orig.BusinessKey
	r.Reverses = &k
	r.Lines = make([]Line, len(orig.Lines))
	for n, l := range orig.Lines {
		r.Lines[n] = l
		if l.Side == Debit {
			r.Lines[n].Side = Credit
		} else {
			r.Lines[n].Side = Debit
		}
	}
	return r, r.Validate()
}

// mirrors reports whether r mirrors o line for line.
func mirrors(o, r Intent) bool {
	if o.Unit != r.Unit || o.Basis != r.Basis || len(o.Lines) != len(r.Lines) {
		return false
	}
	count := map[Line]int{}
	for _, l := range o.Lines {
		m := l
		if l.Side == Debit {
			m.Side = Credit
		} else {
			m.Side = Debit
		}
		count[m]++
	}
	for _, l := range r.Lines {
		count[l]--
	}
	for _, n := range count {
		if n != 0 {
			return false
		}
	}
	return true
}

// Outcome is the answer to one apply.
type Outcome struct {
	Status    result.Status
	Replay    bool
	Reasons   []reasoncodes.Code
	BlockedBy []string
	Detail    string
}

// EvaluateActivation is the answer to any request to post for real. It
// is always BLOCKED: the policies are open and no posting authority
// exists. It never consults a store.
func EvaluateActivation(i Intent) Outcome {
	o := Outcome{Status: result.Blocked, Reasons: []reasoncodes.Code{reasoncodes.PolicyUnresolved},
		BlockedBy: []string{DecisionD41, DecisionOPD06, DecisionOPD08, DecisionOPD09, NoPostingAuthority}}
	if err := i.Validate(); err != nil {
		o.Reasons = append([]reasoncodes.Code{reasoncodes.ContractInvalid}, o.Reasons...)
		o.Detail = err.Error()
	}
	return o
}

// Recorded is one stored intent.
type Recorded struct {
	Intent Intent
	Hash   string
}

// Store is an append-only intent store keyed by business key. Production
// has NO implementation; the synthetic journal and test stores do.
type Store interface {
	// Lookup returns the recorded intent of key, if any.
	Lookup(key string) (Recorded, bool, error)
	// Reversed reports whether key was already reversed.
	Reversed(key string) (bool, error)
	// Append stores rec if key is absent. It returns ErrOutcomeUnknown
	// when the outcome cannot be determined, and inserted=false when the
	// key exists (a concurrent writer won).
	Append(rec Recorded) (inserted bool, err error)
}

// Apply validates i and appends it idempotently to s. It never retries
// blindly: an ErrOutcomeUnknown answer is UNKNOWN, and the caller must
// call Apply again, which looks the key up first (reconcile before retry).
func Apply(s Store, i Intent) Outcome {
	if err := i.Validate(); err != nil {
		return Outcome{Status: result.Failed, Reasons: []reasoncodes.Code{reasoncodes.ContractInvalid}, Detail: err.Error()}
	}
	h := RequestHash(i)
	check := func() (Outcome, bool) {
		prev, ok, err := s.Lookup(i.BusinessKey)
		switch {
		case err != nil:
			return Outcome{Status: result.Unknown, Reasons: []reasoncodes.Code{reasoncodes.OutcomeUnknown}, Detail: err.Error()}, true
		case ok && prev.Hash == h:
			return Outcome{Status: result.Success, Replay: true}, true
		case ok:
			return Outcome{Status: result.Conflict, Reasons: []reasoncodes.Code{reasoncodes.ContractInvalid}, Detail: "business key reused with a different intent"}, true
		}
		return Outcome{}, false
	}
	if o, done := check(); done {
		return o
	}
	if i.Reverses != nil {
		orig, ok, err := s.Lookup(*i.Reverses)
		if err != nil {
			return Outcome{Status: result.Unknown, Reasons: []reasoncodes.Code{reasoncodes.OutcomeUnknown}, Detail: err.Error()}
		}
		if !ok || orig.Intent.Reverses != nil || !mirrors(orig.Intent, i) {
			return Outcome{Status: result.Conflict, Reasons: []reasoncodes.Code{reasoncodes.LinkageConflict}, Detail: "a reversal must mirror an existing original"}
		}
		if done, err := s.Reversed(*i.Reverses); err != nil {
			return Outcome{Status: result.Unknown, Reasons: []reasoncodes.Code{reasoncodes.OutcomeUnknown}, Detail: err.Error()}
		} else if done {
			return Outcome{Status: result.Conflict, Reasons: []reasoncodes.Code{reasoncodes.StaleState}, Detail: "the original is already reversed"}
		}
	}
	inserted, err := s.Append(Recorded{Intent: i, Hash: h})
	switch {
	case errors.Is(err, ErrOutcomeUnknown):
		return Outcome{Status: result.Unknown, Reasons: []reasoncodes.Code{reasoncodes.OutcomeUnknown}, Detail: "reconcile by business key before retry"}
	case err != nil:
		return Outcome{Status: result.Failed, Detail: err.Error()}
	case !inserted:
		if o, done := check(); done {
			return o
		}
		return Outcome{Status: result.Unknown, Reasons: []reasoncodes.Code{reasoncodes.OutcomeUnknown}}
	}
	return Outcome{Status: result.Success}
}
