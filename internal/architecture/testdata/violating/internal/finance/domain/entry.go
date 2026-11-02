// Fixture: a ledger Entry without a business key. Never compiled.
package domain

type Entry struct {
	BusinessKey string
	Basis       string
}

var e = Entry{Basis: "CASH_SHADOW"}
