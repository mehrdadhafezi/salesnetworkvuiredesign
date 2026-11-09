# Slice 15 — Rollback and recovery plan (DRAFT, NOT REHEARSED)

> **R3 update.** Recovery classes, rehearsed failure modes and the proposed RPO / RTO decision matrix
> are in `docs/remediation/r3-disaster-recovery-DRAFT.md` (still DRAFT, not approved). Rows marked "No worker
> process", "Backup and restore NOT VERIFIED" or "no relay process" are superseded by R2 / R3 evidence for
> disposable staging only.

**Principle.** Turning a flag OFF stops **new** target writes. It does **not** reverse committed target
transactions, delivered write-backs, stored evidence or provider effects. No destructive recovery has
been executed. Nothing here has been rehearsed.

## Classes

| Class | Meaning | Reversal mechanism |
|---|---|---|
| **A** | Reversible configuration change | Flag OFF / restore previous value (SQL on the flag store), restart not required |
| **B** | Reversible infrastructure change | Redeploy previous build, re-point routing, restore config |
| **C** | Business effect requiring formal compensation | A new compensating business record approved by the owner. **No compensation command exists in the target today.** |
| **D** | External provider effect requiring reconciliation | Reconcile with the provider by reference, then human resolution with evidence |
| **E** | Irreversible or currently unsupported recovery | Requires a decision; may be impossible |

## Failure modes

| Failure mode | Detection | Immediate action | Recovery path | Class | Current support |
|---|---|---|---|---|---|
| Wrong feature flag enabled | Flag table diff vs change record | Set the flag back | Reconcile commands executed during the window; target rows remain | A (+C for committed effects) | Mechanism exists (SQL); no audited procedure (FF-3) |
| Incomplete migration | `crm-migrate status` shows the failed version | Stop deployment. The failed migration is not recorded as applied. | Fix forward, or restore the DB backup | B / E (no operational down migrations) | Runner stops at first failure (VERIFIED); restore NOT VERIFIED |
| Identity mismatch (wrong person/customer link) | Resolution report, user reports | Disable affected reads/commands | Correct via a new evidence record; never rewrite history | C | Read side only; no correction command |
| Payment duplication | Duplicate submission key / amount report | Flag `payments.manual_write` OFF | Idempotency prevents same-key duplicates (VERIFIED). Cross-system duplicates (legacy + target) need Finance compensation. | C | No compensation command |
| Ledger mismatch | Shadow ledger diff | n/a (no posting) | Fix derivation and rebuild shadow | A (shadow only) | Posting not implemented |
| Missing receipt bytes | Finaliser NOT_RECOVERABLE + reconciliation record | Block review of the attempt | Re-collect evidence from the customer | C | Library only; finaliser not deployed (U6) |
| Worker failure | Backlog age, last-run time | Restart worker | Idempotent steps resume | B | **No worker process or metrics exist** |
| Outbox backlog | Outbox age metric | Scale or restart relay | Inbox dedupe makes redelivery safe (VERIFIED by tests) | B | **No relay process or metric** |
| Provider timeout | Operation in UNKNOWN | Do not retry | Reconcile by reference, then human resolution | D | Library + human resolution route (flag OFF) |
| Biawin double order | Two provider refs for one business key | Stop dispatch | Provider-side cancellation (contract unknown) + compensation | D / E | No connector; provider cancel API unknown |
| Missing callback | Operation ACKNOWLEDGED without confirmation past SLA | — | Lookup/reconcile, then human resolution | D | Library only |
| Wrong service assignment | Supervisor report | Reassign (QUEUED/ASSIGNED only) | IN_PROGRESS reassignment is BLOCKED (OPD-05) | C | Partial |
| Incorrect commission recipient | Parity report | n/a (no posting) | Shadow recompute | A (shadow) | Posting not implemented |
| Inconsistent financial reporting | INV-021/INV-036 parity | Mark metrics NOT FINAL | Rebuild projections (`crm-reporting`) | A | Offline job, manual |
| Authentication outage | Health + login errors | — | n/a (no auth exists) | E today | **No auth** |
| Data corruption | Constraint violations, checksums | Stop writers | Restore from backup + replay verified events | B / E | Backup and restore NOT VERIFIED |
| Partial cross-module execution | Outbox/inbox state mismatch (e.g. activation without case) | Pause relay | Forward recovery: redeliver (idempotent) or compensate; never delete history | C | Relay library verified by fault injection; no process |

## Rehearsal plan (disposable non-production only; not executed)

1. Restore an anonymised real snapshot (after ADR #18) into a disposable environment.
2. For each failure mode above, inject the fault, then record:
   - detection time
   - operator actions
   - reconciliation outcome
   - residual class-C/D items

   Faults to inject include flag misconfiguration, a killed worker, a provider timeout via the synthetic
   adapter, and duplicate submissions.
3. Rehearse backup restore with timing (input to the RPO/RTO decision).
4. Rehearse a module authority switch and switch-back, including drain (runbook §3).
5. Exit criterion: every P0 failure mode has a rehearsed procedure with a named owner.
