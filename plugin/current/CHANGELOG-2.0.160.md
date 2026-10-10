# 2.0.160 — Finance receipt dialog layout

- Stack transactions, receipt galleries and the missing-receipt message vertically, aligned to the top, with independent desktop evidence scrolling.
- Scope readable titles, status summary, review history, rejection form and action layout to the body-mounted receipt modal. Former panel-scoped rules did not reach this dialog.
- Use a single column and one body scroll region at widths up to 800px; keep two metadata columns.
- Preserve gallery images/PDFs and all existing financial actions and permissions. Includes 2.0.159 invoice-review filter fixes.

Validation: source diff limited to public.css and plugin version/build metadata. HTML fixture generated from existing modal/render functions. Browser visual validation could not run because the Chromium executable is unavailable; live WordPress testing has not been performed.
