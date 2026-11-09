# R5 — Management dependency map

**This map invents no approval deadlines.** It shows:
- which decisions block which P0 / P1 items;
- which can be prepared in parallel;
- which must precede implementation or activation.

## 1. Dependency graph

```mermaid
flowchart LR
  subgraph GOV[P-GOV]
    DG[D-03…D-22 record / supersede / void]
  end
  subgraph SEC[P-SEC]
    S1[R5-SEC-01 person identity]
    D34[D-34 provisioning]
    D42[D-42 HR grants]
    A18[ADR #18 real data]
    D45[D-45] 
    D26[D-26]
  end
  subgraph SALES[P-SALES]
    ADR01[ADR-01]
    F01[F01]
    O3[OPD-03]
    O10[OPD-10]
    D32[D-32]
    D28[D-28]
    D29[D-29/31]
    D30[D-30]
  end
  subgraph FIN[P-FIN A–D]
    O8[OPD-08]
    D43[D-43]
    D25[D-25]
    D44[D-44]
    RV[REJECT-VOCAB]
    D41[D-41]
    O6[OPD-06]
    D37[D-37]
    O9[OPD-09]
    D36[D-36]
    O4[OPD-04]
  end
  subgraph HR[P-HR]
    O5[OPD-05]
    SD4[SD-04]
  end
  A18 --> F01
  A18 --> D41
  A18 --> O6
  A18 --> O9
  A18 --> D30
  S1 --> O8
  S1 --> O4
  S1 --> O10
  S1 --> O5
  S1 --> D42
  D34 --> S1
  F01 --> ADR01
  O3 --> ADR01
  O10 --> ADR01
  O8 --> D43
  O8 --> RV
  O8 --> O4
  O6 --> D41
  D41 --> O9
  D36 --> O9
  D37 --> D41
  SD4 --> O5
  O8 --> O5
  ADR01 --> P05((P0-5))
  F01 --> P05
  O3 --> P05
  O5 --> P06((P0-6))
  SD4 --> P06
  O8 --> P07((P0-7))
  O9 --> P07
  D41 --> P07
  O4 --> P07
  O6 --> P07
  A18 --> P03((P0-3 real-data reconciliation))
```

Edge "X → Y" means: X must be decided (or its evidence produced) before Y can be decided meaningfully, or
before Y can be implemented. Both meanings are listed in the tables below.

## 2. Blockers by P0 item

| P0 | Decisions that block it | Must precede implementation | Must precede activation |
|---|---|---|---|
| P0-5 inventory safety | ADR-01, F01, OPD-03, OPD-10, D-32, D-28 | ADR-01, F01 (+ OPD-03 for reset / release) | All of them + P1-8 (legacy writer disable) + runtime evidence |
| P0-6 HR handover | OPD-05, SD-04, R5-SEC-01 | OPD-05, SD-04 | + HR real-account evidence |
| P0-7 financial execution | OPD-08, D-43, REJECT-VOCAB, D-41, OPD-06, OPD-09, D-36, OPD-04, R5-SEC-01 | Per command: OPD-08 (approve / reject), D-41 + OPD-06 (posting), OPD-09 + D-36 (commission), OPD-04 (refund) | + real-data parity (ADR #18) + Finance activation authority |
| P0-3 real-data reconciliation | ADR #18 | ADR #18 | — |
| P0-1 / P0-2 / P0-4 (residual) | D-34, D-42 (P0-1); E-3 (P0-2); column B sign-off (P0-4) | — | Security / Operations sign-off |

**Blocks real staging or reconciliation:** ADR #18 (privacy). Every real-data parity step (F01 coverage,
D-41 parity, OPD-06 source units, OPD-09 engine reconciliation, D-30 resolution report) waits on it.

## 3. What can be prepared in parallel

These packages have no decision dependency on each other and can be reviewed at the same time:
- **P-GOV** (D-03…D-22);
- **ADR #18**;
- **R5-SEC-01 + D-34**;
- **P-REPORT**;
- **P-BIAWIN** (except D-38, which follows D-41);
- **OPD-08 / D-43 / D-25 / D-44**.

## 4. Sequence of management approval meetings (recommendation; no dates)

Grouping keeps the required independent approvals separate:
- Finance, Security and Legal sign their own records;
- the refund package is never folded into payment approval.

| Meeting | Packages / decisions | Required independent sign-offs | Why this order |
|---|---|---|---|
| M1 — Governance and data access | P-GOV (D-03…D-22), ADR #18 | Human Owner; Security + Legal (privacy) | Unblocks every real-data evidence step; clears P1-6 |
| M2 — Identity and access | R5-SEC-01, D-34, D-42, D-45, D-26 | Security owner (separately), HR, Legal (privacy) | SoD for Finance, refunds, delegation and HR handover depends on it |
| M3 — Inventory authority | ADR-01, F01, OPD-03, OPD-10, D-32, D-28, D-29 / D-31, D-30 | Human Owner + Engineering (ADR-01); Sales, MIS, Finance | P0-5; F01 and D-30 may need M1 evidence first |
| M4a — Payment approval | OPD-08, D-43, D-25, D-44, REJECT-VOCAB | Finance; Legal (D-44) | P0-7 approve / reject |
| M4b — Ledger and money | D-41, OPD-06, D-37 | Finance (accounting authority) alone | Needs M1 real-data parity |
| M4c — Entitlements | OPD-09, D-36 | Finance + Product | After M4b (posting basis) |
| M4d — Refunds | OPD-04 | Finance + Product (independent of M4a); Legal for compensation | Separate by rule |
| M5 — Workforce | OPD-05, SD-04 | HR, Sales, Finance, Product | After M2 (person identity) |
| M6 — Services and reporting | P-BIAWIN, P-REPORT, OPD-02, D-40 | Product, Finance, MIS, Sales | Lowest coupling to P0 items |

## 5. Readiness ladder (all six steps are separate)

| Step | Meaning | Status for every OPEN decision today |
|---|---|---|
| 1. Decision evidence available | Facts needed to decide exist | PARTIAL (see each record's "Missing evidence") |
| 2. Ready for human review | Question, options and evidence are documented | YES for records marked READY FOR HUMAN REVIEW; NO for NEEDS FURTHER EVIDENCE |
| 3. Approved by authorized people | Signed record with verifiable evidence | **NO** (0 approvals) |
| 4. Engineering implementation complete | Commands / adapters exist | NO (R4 evaluators only) |
| 5. Runtime evidence verified | Real-environment / real-role evidence | NO |
| 6. Activation separately authorized | Flag or authority switch approved | NO |
