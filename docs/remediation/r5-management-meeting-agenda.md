# R5 — Management meeting agenda (proposal; no dates)

This is a **proposed** agenda. It schedules nothing and approves nothing. The meeting order follows
`r5-management-dependency-map.md` §4.

**Rules for every meeting:**
- **Before the meeting:** every decision under discussion has a DRAFT record in its package document.
- **At the meeting:** a decision is taken only by its named required approvers. A recommendation is never
  a decision.
- **Recording an outcome:** create a v1.0 record from `docs/governance/templates/DECISION-RECORD-TEMPLATE.md`
  with the selected option, the named approvers, the date, the conditions and the evidence of approval
  (minutes or signatures). It is committed to `main` through a reviewed PR. Nothing is recorded on
  anyone's behalf.
- **Implementation and activation:** each is a separate later step with its own authorization.

| # | Meeting | Decisions | Required approvers (independent sign-offs) | Inputs to bring |
|---|---|---|---|---|
| M1 | Governance and data access | P-GOV (D-03…D-22); ADR #18 | Human Owner; Security + Legal | Original Blueprint (if it exists); privacy assessment |
| M2 | Identity and access | R5-SEC-01, D-34, D-42, D-45, D-26 | Security (separately); HR; Legal | HR employee-identity data quality; IdP options |
| M3 | Inventory authority | ADR-01, F01, OPD-03, OPD-10, D-32, D-28, D-29 / D-31, D-30 | Human Owner + Engineering; Sales; MIS; Finance | R4 race-harness evidence; 2.0.176 evidence for D-28 |
| M4a | Payment approval | OPD-08, D-43, D-25, D-44, rejection vocabulary | Finance; Legal | Approval thresholds; reason list |
| M4b | Ledger and money | D-41, OPD-06, D-37 | Finance (accounting authority) | Real-data parity (after M1) |
| M4c | Entitlements | OPD-09, D-36 | Finance + Product | E1–E4 reconciliation on real data (after M1) |
| M4d | Refunds | OPD-04 | Finance + Product; Legal for compensation | Proof standard; execution channel |
| M5 | Workforce handover | OPD-05, SD-04 | HR; Sales; Finance; Product | Open-work impact data |
| M6 | Services and reporting | D-35a–e, D-38, D-33, D-39; OPD-07, D-23, D-24, OPD-01; OPD-02, D-40 | Product; Finance; MIS; Sales | Provider contract; KPI cohort definitions |

## یادداشت فارسی برای جلسات

- در هیچ‌یک از این جلسات تصمیمی از قبل گرفته نشده است. پیشنهادهای مهندسی فقط برای کمک به تصمیم است، نه
  جایگزین آن.
- هر تصمیم فقط با امضای مالک(ها)ی تعیین‌شده معتبر است و باید با صورت‌جلسه یا امضا، تاریخ، گزینهٔ انتخاب‌شده
  و شرایط آن ثبت شود.
- تصمیم‌های مالی (تأیید پرداخت، دفتر کل، کمیسیون، بازپرداخت) جدا از هم و هرکدام با امضای مستقل گرفته می‌شود.
  تصمیم بازپرداخت هرگز با تأیید پرداخت یکی نمی‌شود.
- تصمیم‌های امنیت و هویت را مالک امنیت جداگانه امضا می‌کند.
- پس از تصمیم، پیاده‌سازی و فعال‌سازی هرکدام مجوز جداگانه دارد. سیستم Production همچنان NO-GO است.
