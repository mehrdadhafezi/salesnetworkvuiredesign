/* SN-207 shared accessibility enhancer — loaded by all 8 roles, touches no role logic.
   1) skip link to the workspace landmark; 2) table semantics (th scope, accessible name) re-applied after every render.
   Presentation/semantics only: no data, permission or workflow behaviour. */
(function () {
  'use strict';
  var ws = document.getElementById('ws'); if (!ws) return;
  if (!ws.hasAttribute('tabindex')) ws.setAttribute('tabindex', '-1');
  if (!document.querySelector('.skip-link')) {
    var a = document.createElement('a'); a.className = 'skip-link'; a.href = '#ws'; a.textContent = 'رفتن به محتوای اصلی';
    a.addEventListener('click', function (e) { e.preventDefault(); ws.focus({ preventScroll: false }); });
    document.body.insertBefore(a, document.body.firstChild);
  }
  function name(t) {
    var cap = t.querySelector('caption'); if (cap && cap.textContent.trim()) return;
    if (t.getAttribute('aria-label') || t.getAttribute('aria-labelledby')) return;
    var box = t.closest('.panel,.card,section,.tbl-wrap') || ws, h = box.querySelector('h1,h2,h3,.sec-h h2');
    var txt = h && h.textContent.trim(); if (!txt) { var p = ws.querySelector('h1'); txt = p ? p.textContent.trim() : ''; }
    if (txt) t.setAttribute('aria-label', txt);
  }
  function enhance() {
    ws.querySelectorAll('table').forEach(function (t) {
      t.querySelectorAll('thead th:not([scope])').forEach(function (th) { th.setAttribute('scope', 'col'); });
      name(t);
    });
  }
  var busy = false;
  new MutationObserver(function () { if (busy) return; busy = true; setTimeout(function () { busy = false; enhance(); }, 0); }).observe(ws, { childList: true, subtree: true });
  enhance();
})();
