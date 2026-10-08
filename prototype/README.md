# Seller Panel Redesign — Concept Prototype

Standalone visual prototype. **Not part of the Sales Network plugin**; it uses no plugin files, makes no network calls (except the Vazirmatn web font), and all data in `data.js` is fictional.

## Open it

Any static server works, for example:

```
cd seller-redesign-prototype
python -m http.server 8766
```

Then open `http://127.0.0.1:8766/` (interactive prototype) or `http://127.0.0.1:8766/mockups/` (static mockup gallery). Opening `index.html` directly from disk also works.

## Files

| File | Purpose |
|---|---|
| `index.html` | Shell: header, top tabs, workspace, bottom nav, drawer container |
| `styles.css` | Design tokens and all components (desktop, tablet ≤1100px, mobile ≤760px) |
| `app.js` | View rendering, drawer, menus, demo interactions |
| `data.js` | Mock data |
| `mockups/*.png` | Static renders (1366×768 @1.5x, 390×844 @2x in a phone frame, 768, 1920) |
| `mockups/device.html` | Phone frame used for mobile renders: `device.html?src=view=leads%26queue=all` |

## Deep links (used for the renders)

- `?view=leads&queue=all` — queues: `none`, `no_answer`, `callback`, `not_purchased`, `duplicate`, `invoiced`, `all`
- `?view=leads&queue=all&lead=9043346&outcome=no_answer` — drawer open with an outcome selected
- `?view=invoices&inv=71942055` · `?view=invoices&menu=invoice`
- `?view=conversions&case=104&outcome=customer_declined`
- `?view=wallet` · `?view=behavior` · `?view=request` · `?view=repeat`

## Demo-only

Every submit, save, send, copy, call and filter chip shows a toast and does nothing. Search inputs are visual only. Pagination and the date/product/source filter menus are static.
