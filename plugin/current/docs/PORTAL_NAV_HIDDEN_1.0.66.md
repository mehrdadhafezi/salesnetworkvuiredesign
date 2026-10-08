# 1.0.66 - Portal nav hidden by default

- `[sn_portal_nav]` no longer renders the top portal navigation unless explicitly enabled with `sn_portal_nav_enabled=1` or `[sn_portal_nav show="1"]`.
- New portal pages are created with only their panel shortcode, not the global nav shortcode.
- Unified login remains the source of truth for routing users to the correct panel.
