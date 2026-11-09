# R5 — D-03 … D-22 evidence recovery

**Result: the authoritative content of D-03 … D-22 cannot be recovered from the repository, its full git
history, or the available GitHub pull-request evidence.** Each ID is marked **EVIDENCE NOT RECOVERED —
HUMAN REVIEW REQUIRED**. Nothing was reconstructed or inferred.

## 1. What was searched

| Source | Method | Result |
|---|---|---|
| Current `main` (`50d0f14`), all tracked files | regex `(^\|[^A-Z])D-(0[3-9]\|1[0-9]\|2[0-2])\b` (excluding `OPD-0x`) | Only the **range** "D-03…D-22" (E0 §7B, Arch §32, register, go/no-go). No individual definition. |
| Full git history (all refs, 134 commits) | `git log --all -G` with the same regex | 4 commits (`9d0bc3e`, `4c3ae88`, `560ca6c`, `7f1eb94`). All mention the range only. No individual wording. |
| Naïve substring search `-S "D-0x"` | — | False positives only: matches inside `OPD-03…OPD-10` (e.g. `a0c91ef`). Recorded so a reviewer does not repeat the mistake. |
| GitHub PR #9 … #44 titles | `list_pull_requests` | No PR concerns D-03 … D-22. |
| PR #23 (E0) description and review threads | `pull_request_read` | Description: "D-03…D-26 remain unresolved" (range only). The review thread asked to commit the records or stop citing them. The reply: "the text isn't available in the repo, and inventing it would breach AGENTS.md". |
| `docs/source-of-truth/*` (Persian and English) | grep | No D-03 … D-22. These documents use OPD / SD / F / INV identifiers. |
| Prior audit corpus / "Standalone Implementation Blueprint" | E0 §2: "OWNER-BRIEF … not committed to this repository" | **Not available.** This is the only cited origin of D-03 … D-22 (EV-005). |

## 2. Provenance chronology

| Date (commit) | Source | Statement | Classification |
|---|---|---|---|
| 2026-10-20 (`9d0bc3e`) | E0 §7B, first revision | "D-03…D-22 — As registered in the Blueprint — Per Blueprint" | Historical; the content was delegated to an uncommitted source |
| 2026-10-20 (PR #23 review) | Codex P2 thread `PRRT_kwDOU-nRxs6uZM8x` | "Commit the referenced D-03 through D-22 decision records … or stop citing these IDs as governing gates" | Review finding |
| 2026-10-20 (`4c3ae88`, merged in `add2a36`) | E0 §7B, amended | "Not recorded on main … carry no authority in this repository until committed" | **Authoritative current rule** (EV-004) |
| 2026-10-29 (`560ca6c`) | Architecture V1 §32 | "Content not recorded on main; not authoritative until separately recorded" | Consistent with E0 |
| 2026-11-04 (`7f1eb94`) | Register; go/no-go P1-6; P-GOV package | "Record D-03…D-22 on main, or declare them void" | Consistent |

**Conflict C-01** (`r5-decision-conflict-register.md`): the first E0 revision delegated authority to the
Blueprint, and the amended revision removed that authority. The amended text is the merged one and
governs. The first revision is historical only.

## 3. Per-ID recovery

| ID | Exact source found | Historical date | Commit | Proposal or approved? | Conflicts | Missing provenance | Original question recoverable? | Status |
|---|---|---|---|---|---|---|---|---|
| D-03 | none on `main`, in history, PR bodies or review threads | — | — | unknown (cannot distinguish proposal from approval) | range-level only (C-01) | original wording, date, approver, status | **NO** | **EVIDENCE NOT RECOVERED — HUMAN REVIEW REQUIRED** |
| D-04 | none on `main`, in history, PR bodies or review threads | — | — | unknown (cannot distinguish proposal from approval) | range-level only (C-01) | original wording, date, approver, status | **NO** | **EVIDENCE NOT RECOVERED — HUMAN REVIEW REQUIRED** |
| D-05 | none on `main`, in history, PR bodies or review threads | — | — | unknown (cannot distinguish proposal from approval) | range-level only (C-01) | original wording, date, approver, status | **NO** | **EVIDENCE NOT RECOVERED — HUMAN REVIEW REQUIRED** |
| D-06 | none on `main`, in history, PR bodies or review threads | — | — | unknown (cannot distinguish proposal from approval) | range-level only (C-01) | original wording, date, approver, status | **NO** | **EVIDENCE NOT RECOVERED — HUMAN REVIEW REQUIRED** |
| D-07 | none on `main`, in history, PR bodies or review threads | — | — | unknown (cannot distinguish proposal from approval) | range-level only (C-01) | original wording, date, approver, status | **NO** | **EVIDENCE NOT RECOVERED — HUMAN REVIEW REQUIRED** |
| D-08 | none on `main`, in history, PR bodies or review threads | — | — | unknown (cannot distinguish proposal from approval) | range-level only (C-01) | original wording, date, approver, status | **NO** | **EVIDENCE NOT RECOVERED — HUMAN REVIEW REQUIRED** |
| D-09 | none on `main`, in history, PR bodies or review threads | — | — | unknown (cannot distinguish proposal from approval) | range-level only (C-01) | original wording, date, approver, status | **NO** | **EVIDENCE NOT RECOVERED — HUMAN REVIEW REQUIRED** |
| D-10 | none on `main`, in history, PR bodies or review threads | — | — | unknown (cannot distinguish proposal from approval) | range-level only (C-01) | original wording, date, approver, status | **NO** | **EVIDENCE NOT RECOVERED — HUMAN REVIEW REQUIRED** |
| D-11 | none on `main`, in history, PR bodies or review threads | — | — | unknown (cannot distinguish proposal from approval) | range-level only (C-01) | original wording, date, approver, status | **NO** | **EVIDENCE NOT RECOVERED — HUMAN REVIEW REQUIRED** |
| D-12 | none on `main`, in history, PR bodies or review threads | — | — | unknown (cannot distinguish proposal from approval) | range-level only (C-01) | original wording, date, approver, status | **NO** | **EVIDENCE NOT RECOVERED — HUMAN REVIEW REQUIRED** |
| D-13 | none on `main`, in history, PR bodies or review threads | — | — | unknown (cannot distinguish proposal from approval) | range-level only (C-01) | original wording, date, approver, status | **NO** | **EVIDENCE NOT RECOVERED — HUMAN REVIEW REQUIRED** |
| D-14 | none on `main`, in history, PR bodies or review threads | — | — | unknown (cannot distinguish proposal from approval) | range-level only (C-01) | original wording, date, approver, status | **NO** | **EVIDENCE NOT RECOVERED — HUMAN REVIEW REQUIRED** |
| D-15 | none on `main`, in history, PR bodies or review threads | — | — | unknown (cannot distinguish proposal from approval) | range-level only (C-01) | original wording, date, approver, status | **NO** | **EVIDENCE NOT RECOVERED — HUMAN REVIEW REQUIRED** |
| D-16 | none on `main`, in history, PR bodies or review threads | — | — | unknown (cannot distinguish proposal from approval) | range-level only (C-01) | original wording, date, approver, status | **NO** | **EVIDENCE NOT RECOVERED — HUMAN REVIEW REQUIRED** |
| D-17 | none on `main`, in history, PR bodies or review threads | — | — | unknown (cannot distinguish proposal from approval) | range-level only (C-01) | original wording, date, approver, status | **NO** | **EVIDENCE NOT RECOVERED — HUMAN REVIEW REQUIRED** |
| D-18 | none on `main`, in history, PR bodies or review threads | — | — | unknown (cannot distinguish proposal from approval) | range-level only (C-01) | original wording, date, approver, status | **NO** | **EVIDENCE NOT RECOVERED — HUMAN REVIEW REQUIRED** |
| D-19 | none on `main`, in history, PR bodies or review threads | — | — | unknown (cannot distinguish proposal from approval) | range-level only (C-01) | original wording, date, approver, status | **NO** | **EVIDENCE NOT RECOVERED — HUMAN REVIEW REQUIRED** |
| D-20 | none on `main`, in history, PR bodies or review threads | — | — | unknown (cannot distinguish proposal from approval) | range-level only (C-01) | original wording, date, approver, status | **NO** | **EVIDENCE NOT RECOVERED — HUMAN REVIEW REQUIRED** |
| D-21 | none on `main`, in history, PR bodies or review threads | — | — | unknown (cannot distinguish proposal from approval) | range-level only (C-01) | original wording, date, approver, status | **NO** | **EVIDENCE NOT RECOVERED — HUMAN REVIEW REQUIRED** |
| D-22 | none on `main`, in history, PR bodies or review threads | — | — | unknown (cannot distinguish proposal from approval) | range-level only (C-01) | original wording, date, approver, status | **NO** | **EVIDENCE NOT RECOVERED — HUMAN REVIEW REQUIRED** |

**Note on D-23 … D-45.** These IDs were added later with their content committed (Arch §32, `560ca6c`).
The numbering continues after D-22, but that **does not** imply anything about the content of D-03 … D-22.

## 4. Management package P-GOV — DRAFT — NOT APPROVED

**Question for the Human Owner, for each ID:** formally **record** it (supply the original Blueprint
wording and its approval evidence), **supersede** it (name the replacing decision), or **declare it void**.
No answer is pre-filled. A "record" answer requires the original source document to be committed (or
attached) as evidence. Reconstructing it from memory is not acceptable evidence.

| ID | Recovery status | Choice (owner) | Decided by (name, role) | Date | Evidence of decision | Notes |
|---|---|---|---|---|---|---|
| D-03 | EVIDENCE NOT RECOVERED | ☐ record (supply the original text and approval evidence) ☐ supersede (by: ____ ) ☐ declare void | | | | |
| D-04 | EVIDENCE NOT RECOVERED | ☐ record (supply the original text and approval evidence) ☐ supersede (by: ____ ) ☐ declare void | | | | |
| D-05 | EVIDENCE NOT RECOVERED | ☐ record (supply the original text and approval evidence) ☐ supersede (by: ____ ) ☐ declare void | | | | |
| D-06 | EVIDENCE NOT RECOVERED | ☐ record (supply the original text and approval evidence) ☐ supersede (by: ____ ) ☐ declare void | | | | |
| D-07 | EVIDENCE NOT RECOVERED | ☐ record (supply the original text and approval evidence) ☐ supersede (by: ____ ) ☐ declare void | | | | |
| D-08 | EVIDENCE NOT RECOVERED | ☐ record (supply the original text and approval evidence) ☐ supersede (by: ____ ) ☐ declare void | | | | |
| D-09 | EVIDENCE NOT RECOVERED | ☐ record (supply the original text and approval evidence) ☐ supersede (by: ____ ) ☐ declare void | | | | |
| D-10 | EVIDENCE NOT RECOVERED | ☐ record (supply the original text and approval evidence) ☐ supersede (by: ____ ) ☐ declare void | | | | |
| D-11 | EVIDENCE NOT RECOVERED | ☐ record (supply the original text and approval evidence) ☐ supersede (by: ____ ) ☐ declare void | | | | |
| D-12 | EVIDENCE NOT RECOVERED | ☐ record (supply the original text and approval evidence) ☐ supersede (by: ____ ) ☐ declare void | | | | |
| D-13 | EVIDENCE NOT RECOVERED | ☐ record (supply the original text and approval evidence) ☐ supersede (by: ____ ) ☐ declare void | | | | |
| D-14 | EVIDENCE NOT RECOVERED | ☐ record (supply the original text and approval evidence) ☐ supersede (by: ____ ) ☐ declare void | | | | |
| D-15 | EVIDENCE NOT RECOVERED | ☐ record (supply the original text and approval evidence) ☐ supersede (by: ____ ) ☐ declare void | | | | |
| D-16 | EVIDENCE NOT RECOVERED | ☐ record (supply the original text and approval evidence) ☐ supersede (by: ____ ) ☐ declare void | | | | |
| D-17 | EVIDENCE NOT RECOVERED | ☐ record (supply the original text and approval evidence) ☐ supersede (by: ____ ) ☐ declare void | | | | |
| D-18 | EVIDENCE NOT RECOVERED | ☐ record (supply the original text and approval evidence) ☐ supersede (by: ____ ) ☐ declare void | | | | |
| D-19 | EVIDENCE NOT RECOVERED | ☐ record (supply the original text and approval evidence) ☐ supersede (by: ____ ) ☐ declare void | | | | |
| D-20 | EVIDENCE NOT RECOVERED | ☐ record (supply the original text and approval evidence) ☐ supersede (by: ____ ) ☐ declare void | | | | |
| D-21 | EVIDENCE NOT RECOVERED | ☐ record (supply the original text and approval evidence) ☐ supersede (by: ____ ) ☐ declare void | | | | |
| D-22 | EVIDENCE NOT RECOVERED | ☐ record (supply the original text and approval evidence) ☐ supersede (by: ____ ) ☐ declare void | | | | |

**Effect while undecided:**
- No target behaviour depends on D-03 … D-22 (E0 rule 1).
- Go/no-go blocker **P1-6** stays open.
- If the owner declares them void, no implementation change follows. If the owner records one, its
  content becomes a new decision with its own approval gate.
