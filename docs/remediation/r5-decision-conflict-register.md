# R5 — Decision conflict register

This register lists every place where historical documents, implementation and current governance
disagree. **No historical record is rewritten here.** The Slice 15 assessment (`b3bf31b`) stays as written.
Later engineering (R1–R4) is recorded beside it.

**Precedence used to explain a conflict** (AGENTS.md "Sources of Truth"; this is not a new rule):
1. Cross-Role V1 and Gate-0 (frozen);
2. frozen role product specs;
3. accepted audit / evidence.

Architecture V1, E0 and the Slice 15 register are engineering records beneath those sources. Code and tests
are never authority.

| # | Subject | Source A | Source B | Nature | Current governing text | Resolution needed from | Status |
|---|---|---|---|---|---|---|---|
| C-01 | Authority of D-03 … D-22 | E0 first revision `9d0bc3e`: "as registered in the Blueprint" | E0 amended `4c3ae88` (merged): "no authority until committed" | Historical vs current | The amended E0 (merged) | Human Owner (P-GOV): record / supersede / void | OPEN (decision) |
| C-02 | Owners of OPD decisions | Cross-Role §19 (frozen): e.g. OPD-06 Product + Finance + **Engineering**; OPD-09 Product + Finance + **HR + Engineering**; OPD-02 Product + **Sales + MIS + HR**; OPD-03 **Product** + Sales + MIS + Finance; OPD-08 + MIS policy owner for MIS plans; OPD-10 + MIS source paths | Slice 15 register §1: shorter owner lists ("Product + Finance", "Product", "per §19") | Documentation drift | Cross-Role §19 (frozen source of truth). The R5 records use the §19 owners. | Owners confirm; the register is not edited silently | OPEN (confirmation) |
| C-03 | D-35a–e sub-item mapping | Architecture V1 §14 (`560ca6c`): explicit letters (a) cash credit … (e) conversion mandatory. Architecture V1 §32 row (`560ca6c`): five **unlettered** topics "Subscription+ business model, lifecycle, cash-credit semantics, Biawin fulfilment, mandatory conversion" | Register §1 (`7f1eb94`): unlettered, §14 order. Slice 14 implementation matrix (`1d873a7`): letters assigned from §14 | Mapping conflict inside the frozen architecture (§14 vs §32), carried into the register and the implementation matrix | **None.** Both mappings are preserved below. The implementation matrix is not authority over the architecture, and R5 does not choose between §14 and §32 | Product + Finance confirm the sub-item mapping | **OPEN — UNRESOLVED** pending Product + Finance |
| C-04 | D-34 safe default | Arch §32: "random password + reset flow" | R1: no code writes a credential; provisioning undecided; register marks it "(proposed)" | Documented default not implemented | R1 behaviour: nothing is provisioned | Security + HR (D-34) | OPEN (decision) |
| C-05 | D-28 provenance | Arch §32: "2.0.176 removed the full-payment rule" | Repository contains plugin 2.0.160 only; no 2.0.176 artefact | Missing evidence | 2.0.160 rule kept (safe default) | Owner supplies 2.0.176 evidence before D-28 | OPEN (evidence) |
| C-06 | D-44 unbound-upload expiry | Arch §32 / register: "unbound 7 days" | R2: purge refused outside development; report-only | Documented default narrower in code | R2 behaviour (no purge) | Finance + Legal (D-44) | OPEN (decision) |
| C-07 | D-37 safe default | Arch §32: "entitlement account, flagged provisional" | Slice 14: no installment entitlement exists | Documented default not implemented | No entitlement | Finance (D-37) | OPEN |
| C-08 | D-29 safe default wording | Arch §32: "duplicates as MergeCase hints" | Register: "discovery only; no dedupe" | Wording drift (both forbid merging by phone) | No merge; discovery only (AGENTS.md) | Product + Sales (D-29 / D-31) | OPEN (wording) |
| C-09 | Slice 15 P2-1 / P2-2 | go/no-go §4: tracked 16 MB binary; CI on PRs only | R3: binary removed + CI guard; Go CI on push to `main` + release workflow | Historical vs later engineering | Historical text preserved; R3 evidence supersedes the *engineering facts* only | None (record only) | RESOLVED BY LATER ENGINEERING (recorded) |
| C-10 | Slice 15 P0-1 / P0-2 / P0-4 | go/no-go §4: no auth; workers not deployable; no deployment definition | R1 (auth technically done), R2 (workers implemented, flags OFF), R3 (staging deploy / backup / DR engineering) | Historical vs later engineering | P0-1 / P0-2 / P0-4 **not closed**: provisioning (D-34), production connector (E-3), real infrastructure (column B) remain | Owners and operations | PARTIAL |
| C-11 | Refund separation of duties (R4) | R4 first draft of the refund doc: "requester, reviewer and authorizer must be different people" | R4 review correction `225c53e`: account inequality only; person level fail-closed | Overstatement corrected | `225c53e` | None | RESOLVED |
| C-12 | Code behaviour mistaken for authority | Several R1–R4 docs describe safe defaults as "implemented" | AGENTS.md / E0: only owner records authorize | Interpretation risk | Every R5 record says DRAFT — NOT APPROVED; no safe default is an approval | — | CONTROLLED (consistency test) |

## C-03 in detail: D-35a–e mappings (both preserved, none selected)

Exact text, cited by file version (anchors in `r5-evidence-index.json`: EV-033, EV-046, EV-047, EV-048):

- **Architecture V1 §14** (`docs/implementation/CRM-STANDALONE-TARGET-ARCHITECTURE-V1.md` at `560ca6c`):
  "**ODR D-35 a–e:** (a) cash credit = liability vs voucher vs merchant credit; (b) installment credit =
  financing product?; (c) expiry/spend rules; (d) Biawin as redemption system of record; (e) is conversion
  mandatory for the no-cash path."
- **Architecture V1 §32** (same file and commit), row `D-35a–e`: "Subscription+ business model, lifecycle,
  cash-credit semantics, Biawin fulfilment, mandatory conversion". No letters are given.
- **Slice 15 register §1** (`docs/cutover/decision-approval-register.md` at `7f1eb94`), row `D-35a–e`:
  "Subscription+ model: cash-credit semantics, installment credit, expiry/redemption, Biawin as redemption
  system of record, mandatory conversion" ("per Arch §14"). No letters are given.
- **Slice 14 implementation matrix** (`docs/implementation/slice-14-biawin-command-matrix.md` at `1d873a7`),
  rows `D-35a` … `D-35e`: letters assigned from "Target Arch §14 'ODR D-35 a–e'".

| Sub-item | Arch §14 (explicit letter) | Arch §32 (position in an unlettered list) | Register §1 (position) | Slice 14 matrix (letter) |
|---|---|---|---|---|
| a | cash credit = liability vs voucher vs merchant credit | 1: Subscription+ business model | 1: cash-credit semantics | D-35a cash credit semantics |
| b | installment credit = financing product? | 2: lifecycle | 2: installment credit | D-35b installment credit |
| c | expiry/spend rules | 3: cash-credit semantics | 3: expiry/redemption | D-35c expiry / spend / redemption |
| d | Biawin as redemption system of record | 4: Biawin fulfilment | 4: Biawin as redemption system of record | D-35d Biawin as redemption system of record |
| e | is conversion mandatory for the no-cash path | 5: mandatory conversion | 5: mandatory conversion | D-35e conversion mandatory (no-cash path) |

**What is and is not known:**
- §14 and §32 are in the same frozen document and the same commit. Neither says that it overrides the other.
- §32 gives no letters. Reading it positionally (column 3) is an interpretation; a non-positional reading
  (a summary list of topics) is also possible. The text does not decide which.
- The topics also differ: §32 names "business model", "lifecycle" and "Biawin fulfilment"; §14 names
  "installment credit" and "Biawin as redemption system of record".
- The Slice 14 matrix is the most detailed source, but it is an **implementation** document. Under the
  source-of-truth hierarchy (AGENTS.md) it does not decide a product mapping. Code that follows it is
  implementation behaviour, not authority (C-12).

**R5 handling:** every D-35a–e record is titled neutrally ("Sub-item (x): mapping UNRESOLVED (C-03)"). It
quotes both mappings, depends on the C-03 confirmation, and has readiness "NEEDS PRODUCT + FINANCE MAPPING
CONFIRMATION (C-03) BEFORE A DECISION". Its alternatives are listed under the §14 labelling only so that
they can be read, and are void if a different mapping is confirmed. No record states or implies that a
mapping is approved.

## Historical vs current (summary)

| Item | Slice 15 (historical, `b3bf31b`) | After R1–R4 (engineering) | Business decision status |
|---|---|---|---|
| P0-1 auth | No auth | Technical auth done (R1) | D-34 / D-42 / R5-SEC-01 OPEN |
| P0-2 workers / write-back | Not deployable | Workers implemented, flags OFF (R2) | E-3 OPEN |
| P0-3 real-data reconciliation | Not run | Not run | ADR #18 OPEN |
| P0-4 infrastructure | Not defined | Staging engineering (R3) | Column B unsigned |
| P0-5 inventory safety | Undecided, not implemented | Evaluators + race harness (R4) | ADR-01, F01, OPD-03 OPEN |
| P0-6 HR handover | Open | Unchanged | OPD-05, SD-04 OPEN |
| P0-7 financial execution | Open | Contracts (R4) | OPD-08, OPD-09, D-41, OPD-04, OPD-06 OPEN |
