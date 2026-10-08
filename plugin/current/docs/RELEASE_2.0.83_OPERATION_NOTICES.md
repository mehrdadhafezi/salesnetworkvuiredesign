# Release 2.0.83 — Operation result notices

- Customer and Operations panel notices display only «موفق» or «ناموفق».
- Added an accessible close button and a five-second automatic dismissal for both locations, including a notice shown inside a reopened customer card.
- After rendering, result query parameters are removed from browser history, so refreshing the page does not repeat the notification.
- New redirects carry only the result status flag, without embedding detailed operation messages in the URL.

Verification: syntax checks passed for both updated JavaScript files; the plugin static audit passed. PHP CLI and a live WordPress runtime are unavailable here.
