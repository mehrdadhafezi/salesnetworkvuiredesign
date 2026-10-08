# Release 2.0.44 — Operations Role Consolidation and Converter Save

## Scope

- `sn_project_manager` and `sn_project_expert` are no longer created or offered as active organizational roles.
- Existing users and HR profiles in those legacy roles are migrated respectively to `operations_sales_manager` and `operations_sales_expert`.
- Legacy project pages and shortcodes remain read-compatible aliases to the Operations Sales panels, so existing WordPress pages do not break after upgrade.
- Historical project tables and identifiers are retained; this release does not delete operational history.

## Card routing contract

For every selected content/product card, the settings page requires a default Operations Sales Manager. Administrators may also add any number of branch-specific routes:

`content card + originating Sales Manager -> destination Operations Sales Manager`

The resolver first checks the exact originating Sales Manager route. If no exact route exists, it uses the card default. The chosen destination is snapshotted on the created membership so later configuration changes do not rewrite history.

The Operations Sales hierarchy remains:

`Operations Sales Manager -> Operations Sales Supervisor -> Operations Sales Expert`

After upgrading accounts that previously used a direct Project Manager -> Project Expert hierarchy, HR should review and assign the appropriate Operations Sales Supervisor between those levels.

## Converter save fix

The «ذخیره نتیجه تماس» button is now always actionable. Conditional inputs are still enforced by the existing form validation and AJAX/server handler:

- accepted outcomes require their product/next-step fields;
- callback/follow-up outcomes require a valid follow-up date when applicable;
- declined outcomes require the configured decline reason;
- nonce, login, ownership/scope and server validation remain unchanged.

This prevents a stale client-side state from permanently leaving the submit button disabled while retaining validation feedback.

## Upgrade acceptance checklist

1. Back up the database and install the ZIP on Staging.
2. Confirm plugin version `2.0.44` and open a normal admin request so upgrade routines run.
3. In HR, confirm legacy Project Manager users are Operations Sales Managers and legacy Project Experts are Operations Sales Experts.
4. Review migrated experts and assign each through the required Operations Sales Supervisor hierarchy.
5. In Sales Network settings, select a card, its default Operations Sales Manager, and two branch-specific Sales Manager routes; save and reload.
6. Sell the same card once through each source branch and verify each membership reaches the configured destination manager.
7. Test a branch without an exact route and verify the card default is used.
8. In the converter panel, test accepted, follow-up and declined outcomes. Confirm the button can be clicked and missing fields produce a validation message.
9. Smoke-test invoice/payment, wallet/commission, chat and existing historical project records.

## Local verification

- JavaScript syntax checks pass.
- The packaged static audit passes for version `2.0.44`.
- The ZIP integrity test passes.
- PHP CLI, WordPress and MySQL are not available in the build environment; PHP lint and runtime acceptance remain required on Staging before Production.
