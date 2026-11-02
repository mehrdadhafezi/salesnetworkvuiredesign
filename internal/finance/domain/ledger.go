// Package domain is the finance SHADOW ledger model (Architecture V1 §16,
// Slice 9). Everything here is a derived shadow fact: not the financial
// system of record, not settled, not bank-confirmed, not Finance-approved.
//
// Invariants enforced here (and again by the database):
//   - every entry has a deterministic business key, unique in its run;
//   - every entry is single-unit and balanced: DEBITS == CREDITS in that
//     unit (cross-unit balancing is forbidden: an entry with lines in two
//     units is rejected even if each unit balances);
//   - amounts are int64 in an explicit unit (IRR | IRT); no float, no
//     conversion, no rounding (OPD-06 unresolved);
//   - a correction is a new entry with reverses_entry_id pointing to an
//     earlier entry of the same run; an entry is reversed at most once and
//     never by itself; the original is never mutated.
package domain

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/money"
)

// DerivationVersion is the shadow derivation version. Changing what is
// derived (accounts, keys, bases, rules) requires a new version; runs of
// different versions coexist and are never rewritten.
const DerivationVersion = 1

// RecordKind marks every Slice 9 record.
const RecordKind = "SHADOW"

// Bases (D-41 unresolved: both are computed where evidence permits, neither
// is canonical, neither is an official / accounting balance).
const (
	AccrualShadow = "ACCRUAL_SHADOW"
	CashShadow    = "CASH_SHADOW"
	// WalletMirrorShadow mirrors legacy wallet postings. It is neither an
	// accrual nor a cash basis: D-41 does not apply to it.
	WalletMirrorShadow = "WALLET_MIRROR_SHADOW"
)

// Bases lists every basis in display order. No basis is preferred.
func Bases() []string { return []string{AccrualShadow, CashShadow, WalletMirrorShadow} }

// Sides.
const (
	DebitSide  = "DEBIT"
	CreditSide = "CREDIT"
)

// Account categories, instantiated only where Slice 9 evidence exists.
const (
	CustomerReceivable    = "customer_receivable"
	RevenueByProductKind  = "revenue_by_product_kind"
	CashBankClearing      = "cash_bank_clearing"
	CommissionPayable     = "commission_payable"
	LegacyWalletUnclassed = "legacy_wallet_unclassified"
	LegacyWalletContra    = "legacy_wallet_contra"
)

// Categories lists the account categories with their normal side.
var Categories = map[string]string{
	CustomerReceivable:    DebitSide,
	RevenueByProductKind:  CreditSide,
	CashBankClearing:      DebitSide,
	CommissionPayable:     CreditSide,
	LegacyWalletUnclassed: CreditSide,
	LegacyWalletContra:    DebitSide,
}

// Entry evidence status.
const (
	EntryConsistent           = "CONSISTENT"
	EntryConflict             = "CONFLICT"
	EntryUnresolvedCorrection = "UNRESOLVED_CORRECTION"
)

// FinanceReview of every Slice 9 entry: nothing is Finance-reviewed. A
// gateway auto-approval keeps its GATEWAY_AUTO provenance.
const NotFinanceReviewed = "NOT_FINANCE_REVIEWED"

// Time evidence of an entry.
const (
	TimeAbsent         = "BUSINESS_TIME_ABSENT"
	TimeZoneUnasserted = "SOURCE_TIME_ZONE_UNASSERTED"
)

// Errors.
var (
	ErrUnbalanced     = errors.New("finance: entry is not balanced (debits != credits)")
	ErrCrossUnit      = errors.New("finance: cross-unit entry is forbidden")
	ErrInvalidEntry   = errors.New("finance: invalid ledger entry")
	ErrDuplicateKey   = errors.New("finance: duplicate business key")
	ErrInvalidReverse = errors.New("finance: invalid reversal")
)

// Account is one ledger account of a run. Key is its stable identity: a
// category plus a stable target / native reference; never a phone, name or
// email.
type Account struct {
	Key, Category             string
	SubjectDomain, SubjectRef string
	CustomerLink, CustomerID  *string
	HolderKey, HolderUserID   *string
	WalletTypeRaw, Provenance *string
	Route                     *string
}

// NormalSide returns the category's normal side.
func (a Account) NormalSide() string { return Categories[a.Category] }

// SourceRef is one source reference of an entry.
type SourceRef struct {
	Domain string `json:"domain"`
	Ref    string `json:"ref"`
	Role   string `json:"role"`
}

// Line is one ledger line.
type Line struct {
	AccountKey string
	Side       string
	Amount     int64
	SourceRef  *string
}

// Entry is one shadow ledger entry.
type Entry struct {
	BusinessKey   string
	Basis         string
	Unit          money.Unit
	EffectKind    string
	Provenance    string
	SourceRefs    []SourceRef
	SourceTimeRaw *string
	TimeEvidence  string
	// ReversesKey is the business key of the reversed entry (nil = none).
	ReversesKey *string
	Status      string
	Lines       []Line
}

var businessKeyPattern = regexp.MustCompile(`^[a-z_]+(/[A-Za-z0-9_.:-]+){4,}/v[1-9][0-9]*$`)

// BusinessKey builds the deterministic key
// source-domain / subject-ref / effect-kind / unit / basis / version.
func BusinessKey(domain, kind, subject, effect, unit, basis string) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s/%s/v%d", domain, kind, subject, effect, unit, basis, DerivationVersion)
}

// Validate checks one entry in isolation.
func (e Entry) Validate() error {
	if !businessKeyPattern.MatchString(e.BusinessKey) || len(e.BusinessKey) > 300 {
		return fmt.Errorf("%w: business key %q", ErrInvalidEntry, e.BusinessKey)
	}
	switch e.Basis {
	case AccrualShadow, CashShadow, WalletMirrorShadow:
	default:
		return fmt.Errorf("%w: basis %q", ErrInvalidEntry, e.Basis)
	}
	if !e.Unit.Valid() {
		return fmt.Errorf("%w: unit", ErrInvalidEntry)
	}
	if len(e.SourceRefs) == 0 || e.EffectKind == "" || e.Provenance == "" {
		return fmt.Errorf("%w: source refs / effect / provenance required", ErrInvalidEntry)
	}
	switch e.Status {
	case EntryConsistent, EntryConflict, EntryUnresolvedCorrection:
	default:
		return fmt.Errorf("%w: status %q", ErrInvalidEntry, e.Status)
	}
	if e.ReversesKey != nil && *e.ReversesKey == e.BusinessKey {
		return fmt.Errorf("%w: an entry cannot reverse itself", ErrInvalidReverse)
	}
	return Balanced(e.Unit, e.Lines)
}

// Balanced checks the double-entry invariant for one single-unit entry:
// at least one debit and one credit, positive amounts, DEBITS == CREDITS.
func Balanced(unit money.Unit, lines []Line) error {
	var dr, cr int64
	var nd, nc int
	for _, l := range lines {
		if l.Amount <= 0 || l.AccountKey == "" {
			return fmt.Errorf("%w: line amount must be positive with an account", ErrInvalidEntry)
		}
		switch l.Side {
		case DebitSide:
			if dr > math.MaxInt64-l.Amount {
				return fmt.Errorf("%w: %w", ErrInvalidEntry, money.ErrOverflow)
			}
			dr += l.Amount
			nd++
		case CreditSide:
			if cr > math.MaxInt64-l.Amount {
				return fmt.Errorf("%w: %w", ErrInvalidEntry, money.ErrOverflow)
			}
			cr += l.Amount
			nc++
		default:
			return fmt.Errorf("%w: side %q", ErrInvalidEntry, l.Side)
		}
	}
	if nd == 0 || nc == 0 || dr != cr {
		return ErrUnbalanced
	}
	return nil
}

// UnitLine is a line with its own unit, as a caller might propose it.
type UnitLine struct {
	Line
	Unit money.Unit
}

// SingleUnit returns the one unit of the lines, or ErrCrossUnit.
func SingleUnit(lines []UnitLine) (money.Unit, error) {
	if len(lines) == 0 {
		return 0, ErrInvalidEntry
	}
	u := lines[0].Unit
	for _, l := range lines[1:] {
		if l.Unit != u {
			return 0, ErrCrossUnit
		}
	}
	if !u.Valid() {
		return 0, ErrInvalidEntry
	}
	return u, nil
}

// Journal is one run's append-only journal under construction.
type Journal struct {
	Accounts map[string]Account
	Entries  []Entry
	byKey    map[string]int
	reversed map[string]bool
}

// NewJournal returns an empty journal.
func NewJournal() *Journal {
	return &Journal{Accounts: map[string]Account{}, byKey: map[string]int{}, reversed: map[string]bool{}}
}

// Account registers (or returns) an account by key.
func (j *Journal) Account(a Account) string {
	if _, ok := j.Accounts[a.Key]; !ok {
		j.Accounts[a.Key] = a
	}
	return a.Key
}

// Add appends one validated entry. Duplicate business keys, unknown
// accounts, reversals of unknown / later / already reversed entries are
// rejected; nothing already appended is ever changed.
func (j *Journal) Add(e Entry) error {
	if err := e.Validate(); err != nil {
		return err
	}
	if _, dup := j.byKey[e.BusinessKey]; dup {
		return fmt.Errorf("%w: %s", ErrDuplicateKey, e.BusinessKey)
	}
	for _, l := range e.Lines {
		if _, ok := j.Accounts[l.AccountKey]; !ok {
			return fmt.Errorf("%w: unknown account %q", ErrInvalidEntry, l.AccountKey)
		}
	}
	if e.ReversesKey != nil {
		i, ok := j.byKey[*e.ReversesKey]
		if !ok {
			return fmt.Errorf("%w: reversed entry %q does not exist", ErrInvalidReverse, *e.ReversesKey)
		}
		if j.reversed[*e.ReversesKey] {
			return fmt.Errorf("%w: entry %q is already reversed", ErrInvalidReverse, *e.ReversesKey)
		}
		if j.Entries[i].Unit != e.Unit {
			return fmt.Errorf("%w: %w", ErrInvalidReverse, ErrCrossUnit)
		}
		j.reversed[*e.ReversesKey] = true
	}
	j.byKey[e.BusinessKey] = len(j.Entries)
	j.Entries = append(j.Entries, e)
	return nil
}

// Entry returns the entry with the business key.
func (j *Journal) Entry(key string) (Entry, bool) {
	i, ok := j.byKey[key]
	if !ok {
		return Entry{}, false
	}
	return j.Entries[i], true
}

// Reversed reports whether the entry was reversed.
func (j *Journal) Reversed(key string) bool { return j.reversed[key] }

// Balance is the per-unit balance of one account in one basis, signed on
// the account's normal side.
type Balance struct {
	Debits, Credits int64
	Overflow        bool
}

// Net is the balance on the normal side; ok false on overflow.
func (b Balance) Net(normal string) (int64, bool) {
	if b.Overflow {
		return 0, false
	}
	a, c := b.Debits, b.Credits
	if normal == CreditSide {
		a, c = c, a
	}
	if (c > 0 && a < math.MinInt64+c) || (c < 0 && a > math.MaxInt64+c) {
		return 0, false
	}
	return a - c, true
}

// AccountBalance sums the account's lines in one basis and unit. A balance
// is always derived from lines; it is never stored as mutable state.
func (j *Journal) AccountBalance(account, basis string, unit money.Unit) Balance {
	var b Balance
	for _, e := range j.Entries {
		if e.Basis != basis || e.Unit != unit {
			continue
		}
		for _, l := range e.Lines {
			if l.AccountKey != account {
				continue
			}
			p := &b.Credits
			if l.Side == DebitSide {
				p = &b.Debits
			}
			if *p > math.MaxInt64-l.Amount {
				b.Overflow = true
				continue
			}
			*p += l.Amount
		}
	}
	return b
}

// Parsed is one exactly parsed amount.
type Parsed struct {
	Amount *int64
	Unit   *money.Unit
	Repr   string // REPRESENTABLE / UNREPRESENTABLE / ABSENT
}

var decimalPattern = regexp.MustCompile(`^(-?)([0-9]{1,19})(?:\.([0-9]{1,6}))?$`)

// ParseExact parses a raw legacy decimal in its unit exactly: a non-zero
// fractional part, a missing / unknown unit or an int64 overflow is
// UNREPRESENTABLE (never rounded, never converted); empty is ABSENT (never
// zero). A leading minus is kept (legacy balances may be negative).
func ParseExact(raw *string, unit *string) Parsed {
	if raw == nil || *raw == "" {
		return Parsed{Repr: "ABSENT"}
	}
	var u *money.Unit
	if unit != nil {
		if pu, err := money.ParseUnit(*unit); err == nil {
			u = &pu
		}
	}
	m := decimalPattern.FindStringSubmatch(*raw)
	if m == nil || u == nil {
		return Parsed{Unit: u, Repr: "UNREPRESENTABLE"}
	}
	for _, c := range m[3] {
		if c != '0' {
			return Parsed{Unit: u, Repr: "UNREPRESENTABLE"}
		}
	}
	n, err := strconv.ParseInt(m[1]+m[2], 10, 64)
	if err != nil {
		return Parsed{Unit: u, Repr: "UNREPRESENTABLE"}
	}
	return Parsed{Amount: &n, Unit: u, Repr: "REPRESENTABLE"}
}
