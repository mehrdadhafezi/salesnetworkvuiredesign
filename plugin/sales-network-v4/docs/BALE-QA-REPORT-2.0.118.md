# QA Report — Bale/Safir CRM Integration 2.0.118

## Automated checks passed

- PHP syntax lint: all plugin PHP files passed.
- CRM → Gateway HMAC canonical/signature test: PASS.
- Gateway → CRM HMAC verification test: PASS.
- Replay nonce rejection: PASS.
- Notification request-id determinism/idempotency: PASS.
- Secret/Mock static scan: no runtime Bot Token, Safir API Key or `CRM_MOCK_MODE=true` embedded in plugin.
- Existing SMS implementation remains present; Bale is an additive channel.

## Source coverage reviewed

Bale mirror hooks were added to the active SMS paths in 2.0.117, including invoice creation/stages/resend, payment completion, Dot flow, Customer Portal OTP/invite, HR credentials/password, Operations upgrade/wallet and payment reward.

Admin SMS test and SMS preview remain SMS-specific by design; a separate Bale/Safir test action is provided in the Bale settings tab.

## Not executed in build environment

A real Safir delivery cannot be executed during package build because Production `BALE_SAFIR_API_KEY`, Bot Token and CRM/Gateway shared secret are intentionally not embedded in the package. Run the Production Checklist after installation.
