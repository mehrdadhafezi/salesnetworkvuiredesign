# R4 — Ledger posting contract

**No financial posting exists.** `internal/finance/posting` defines and validates a posting *intent* and
proves its invariants on synthetic stores only.
- **No production authority:** there is no production store, route, command or posting authority.
- **Not wired:** lint `r4-evaluator-wiring` keeps the package importable only by tests.
- **Always BLOCKED:** `EvaluateActivation` always answers BLOCKED by D-41, OPD-06, OPD-08, OPD-09 and
  `NO_POSTING_AUTHORITY`.
- **Shadow ledger:** the Slice 9 shadow ledger (`internal/finance/domain`) is unchanged and remains a
  shadow.

## Contract

| Element | Rule |
|---|---|
| Posting intent | `Intent{BusinessKey, Basis, Unit, Lines, Source, EffectiveAt, RecordedAt, Reverses}` |
| Stable business key | Pattern `domain\|part\|part\|part[...]` (3–8 parts); identifies one economic effect |
| Debit / credit | ≥ 2 lines; side `DEBIT` / `CREDIT`; account names validated (no chart of accounts is invented) |
| Balanced | Σ debits = Σ credits, overflow-checked (int64) |
| Explicit unit | One unit per intent (IRR or IRT); amounts are positive int64 minor units. No float, conversion or rounding (OPD-06) |
| Basis | Must be named (`ACCRUAL` or `CASH`). **Neither is chosen** (D-41); activation is BLOCKED for both |
| Idempotency | The same key and the same canonical request hash is a replay (no new effect); the same key with a different request is CONFLICT |
| Provenance | Source system plus at least one source invoice or payment reference |
| Posting vs effective time | Both explicit and UTC; they are distinct fields |
| Reversal | A separate compensating intent (`Reversal`): mirrored lines, same unit and basis, its own key. The original is never mutated or deleted; at most one reversal; a reversal is never reversed again |
| UNKNOWN outcome | A store reporting `ErrOutcomeUnknown` → UNKNOWN. A retry first looks the key up (reconcile before retry), so there is never a blind second effect |
| Conflict | Key reuse, a non-mirroring or missing original, or an already-reversed original |

## Evidence

**Unit tests** (`posting_test.go`):
- invalid intents: unbalanced, zero or negative, no unit, no or unknown basis, no provenance, bad key,
  one line, bad side, zero or non-UTC time, overflow;
- activation is always BLOCKED;
- replays never change the journal;
- key reuse → CONFLICT;
- the reversal nets to zero, and a second reversal → CONFLICT;
- the original is unchanged;
- no cross-unit leakage;
- a lost acknowledgement → UNKNOWN, then the retry is a replay with one record.

**PostgreSQL** (`posting_pg_test.go`, disposable database, synthetic schema `syn_ledger`, test-only
store):
- 20 concurrent identical requests → exactly 1 SUCCESS + 19 replays, 1 posting row, 2 line rows, net
  1 000;
- key reuse → CONFLICT;
- a lost acknowledgement → UNKNOWN, reconciled to a replay;
- reversal → a separate row, net 0 for that effect; a second reversal → CONFLICT;
- UPDATE and DELETE on the posting and line rows are refused (append-only triggers).

**Self-check during development:** a defect in the first test store wrote a line row before checking
the key insert. The test reported `lines 3 net 2000`, i.e. it detected a duplicate economic effect. The
store was fixed, and the defect is recorded here as evidence that the test is sensitive.

## Not decided here

- D-41 (accrual vs cash);
- OPD-06 (units, rounding, the 0.5 boundary);
- OPD-08 (who approves);
- OPD-09 (entitlement routing);
- the chart of accounts;
- posting time vs effective-time policy;
- the reconciliation owner.

All are required before any posting authority exists.
