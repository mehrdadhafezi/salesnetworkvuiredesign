# Release 2.0.119 — Seller invoice Bale checkbox

- Seller simple pre-invoice form now shows delivery channels explicitly.
- SMS remains canonical and always enabled for this flow.
- New **ارسال در بله** checkbox is enabled by default.
- When checked: invoice link is sent by SMS and mirrored to Bale/Safir.
- When unchecked: invoice issuance and SMS continue normally, Bale is skipped for that invoice only.
- Server accepts `send_bale=0|1`; missing keeps 2.0.118 compatibility and requests Bale.
- Seller success notice reports SMS and Bale delivery separately.
