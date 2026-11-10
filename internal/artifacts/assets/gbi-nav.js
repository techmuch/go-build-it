// Added by go-build-it to the HTML it shows in its document viewer (FR-144, ADR-014).
// It tells the viewer where the page is, so the viewer can switch tabs and keep the
// address bar current, and it opens Markdown links rendered. It does nothing when the
// page is not inside go-build-it's viewer.
(function () {
  'use strict';
  if (window.parent === window) return;
  // ?gbi-view=v02 shows only that view (Compare, FR-147): everything around it is hidden.
  const only = new URLSearchParams(location.search).get('gbi-view');
  if (only) {
    const isolate = () => {
      const el = document.getElementById(only) || document.querySelector('[data-view="' + only.replace(/"/g, '') + '"]');
      if (!el) return;
      for (let n = el; n && n !== document.body; n = n.parentElement) {
        for (const sib of Array.from(n.parentElement ? n.parentElement.children : [])) if (sib !== n && sib.tagName !== 'SCRIPT' && sib.tagName !== 'STYLE') sib.style.display = 'none';
      }
      el.style.margin = '0';
      window.scrollTo(0, 0);
    };
    if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', isolate); else isolate();
  }
  const goTo = (hash) => {
    if (!hash) return;
    const el = document.getElementById(decodeURIComponent(hash.slice(1)));
    if (!el) return;
    el.scrollIntoView({ block: 'start' });
    document.querySelectorAll('.gbi-target').forEach((x) => x.classList.remove('gbi-target'));
    el.classList.add('gbi-target');
  };
  const st = document.createElement('style');
  st.textContent = '.gbi-target{background:#fff8e6!important;outline:2px solid #f1d9a3;outline-offset:2px;border-radius:3px}';
  document.head.appendChild(st);
  const post = () => window.parent.postMessage({ gbi: 'nav', href: location.href, path: location.pathname, hash: location.hash, title: document.title }, '*');
  window.addEventListener('load', () => { goTo(location.hash); post(); });
  window.addEventListener('hashchange', () => { goTo(location.hash); post(); });
  document.addEventListener('gbi-md-rendered', () => { goTo(location.hash); post(); });
  document.addEventListener('click', (e) => {
    const a = e.target && e.target.closest ? e.target.closest('a[href]') : null;
    if (!a || e.defaultPrevented || e.button !== 0 || e.metaKey || e.ctrlKey) return;
    const u = new URL(a.getAttribute('href'), location.href);
    if (u.host !== location.host) { a.target = '_blank'; a.rel = 'noopener'; return; }
    if (/\.md$/i.test(u.pathname) && !u.searchParams.has('gbi')) {
      e.preventDefault();
      u.searchParams.set('gbi', 'md');
      location.href = u.pathname + u.search + u.hash;
    }
  });
  window.addEventListener('message', (e) => {
    if (e.source === window.parent && e.data && e.data.gbi === 'hash') {
      if (location.hash === e.data.hash) goTo(e.data.hash); else location.hash = e.data.hash || '';
    }
  });
})();
