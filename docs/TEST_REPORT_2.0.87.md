# Test report — 2.0.87 no-answer tab routing hotfix

## Scope

This release is based on 2.0.86 and preserves the card-payment access hotfix. It changes only seller stage-one status filtering/routing plus static regression checks and release documentation.

## Static verification

- Plugin header and `SN_VERSION`: `2.0.87`.
- `node --check` passes for the static audit and all public JavaScript files through the project audit.
- `node tools/static-audit.js`: PASS.
- PHP lint: 22 plugin PHP files PASS (reported by the static audit).
- Duplicate PHP methods: 0.
- Seller stage-one statuses remain exactly four.
- New `noAnswerCanonicalRouting` regression gate: PASS.

## No-answer cross-panel audit

- Seller: canonical filter slug is used; server normalizes legacy Persian filter values.
- Converter/self-conversion: existing canonical `no_answer` path verified.
- Operations execution: existing canonical `no_answer` path verified.
- Project workflow: existing canonical `no_answer` path verified.
- Sales-manager archive: existing `archive_reason='no_answer'` path verified.

## Safety

- No schema migration.
- No delete/truncate/bulk rewrite.
- No role/capability widening.
- No payment-flow change beyond the already included 2.0.86 hotfix.
- Existing manager archive policy is unchanged: third no-answer attempt + three days inactivity.

## Staging acceptance still required

1. Legacy seller lead -> select `جواب نداده` -> row must immediately appear under the seller `جواب نداده` tab.
2. MIS/V4 distribution item -> same test.
3. Record attempts 2 and 3 and verify the counter.
4. Verify `تماس مجدد` and `عدم خرید` tabs still route correctly.
5. Verify converter, Operations, and manager archive panels still save/display no-answer states.
