// <gbi-md src="file.md"> renders a Markdown file inside an HTML page (FR-145, ADR-014).
// Served by go-build-it at /gbi-assets/gbi-md.js. Requirement lines ("**FR-07**: …") and
// headings become anchors, so requirements.html#FR-07 works.
(function () {
  'use strict';
  const esc = (s) => s.replace(/[&<>"]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]));
  const slug = (s) => s.toLowerCase().replace(/<[^>]+>/g, '').replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '');
  const reqID = /^\*\*((?:FR|NFR)-\d+[a-z]?)\*\*/;

  function inline(s, base) {
    const codes = [];
    s = s.replace(/`([^`]+)`/g, (_, c) => { codes.push(c); return '\u0000' + (codes.length - 1) + '\u0000'; });
    s = esc(s);
    s = s.replace(/!\[([^\]]*)\]\(([^)\s]+)\)/g, (_, a, u) => '<img alt="' + a + '" src="' + fix(u, base) + '">');
    s = s.replace(/\[([^\]]+)\]\(([^)\s]+)\)/g, (_, t, u) => '<a href="' + fix(u, base) + '">' + t + '</a>');
    s = s.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>').replace(/(^|[^*])\*([^*\s][^*]*)\*/g, '$1<em>$2</em>').replace(/(^|\W)_([^_]+)_(?=\W|$)/g, '$1<em>$2</em>');
    return s.replace(/\u0000(\d+)\u0000/g, (_, i) => '<code>' + esc(codes[+i]) + '</code>');
  }
  // Links in the Markdown are relative to the .md file, not to the page that includes it.
  function fix(u, base) {
    if (/^([a-z]+:|#|\/)/i.test(u)) return u;
    try { const abs = new URL(u, base); return abs.pathname + abs.search + abs.hash; } catch (e) { return u; }
  }

  function render(md, base) {
    const lines = md.replace(/\r\n?/g, '\n').split('\n');
    const out = [];
    let i = 0;
    const para = [];
    const flush = () => { if (para.length) { const t = para.join(' '); const m = t.match(reqID); out.push('<p' + (m ? ' id="' + m[1] + '" class="req"' : '') + '>' + inline(t, base) + '</p>'); para.length = 0; } };
    while (i < lines.length) {
      const l = lines[i];
      let m;
      if (/^```/.test(l)) {
        flush(); const code = []; i++;
        while (i < lines.length && !/^```/.test(lines[i])) code.push(lines[i++]);
        out.push('<pre><code>' + esc(code.join('\n')) + '</code></pre>'); i++; continue;
      }
      if ((m = l.match(/^(#{1,6})\s+(.*)$/))) {
        flush(); const h = inline(m[2], base); const id = (m[2].match(/^((?:FR|NFR)-\d+[a-z]?)\b/) || [])[1] || slug(m[2]);
        out.push('<h' + m[1].length + ' id="' + id + '">' + h + '</h' + m[1].length + '>'); i++; continue;
      }
      if (/^\s*(---|\*\*\*|___)\s*$/.test(l)) { flush(); out.push('<hr>'); i++; continue; }
      if (/^\s*>/.test(l)) {
        flush(); const q = [];
        while (i < lines.length && /^\s*>/.test(lines[i])) q.push(lines[i++].replace(/^\s*>\s?/, ''));
        out.push('<blockquote>' + render(q.join('\n'), base) + '</blockquote>'); continue;
      }
      if (/^\s*\|.*\|\s*$/.test(l) && i + 1 < lines.length && /^\s*\|?[\s:-]+\|[\s|:-]*$/.test(lines[i + 1])) {
        flush(); const cells = (r) => r.trim().replace(/^\||\|$/g, '').split('|').map((c) => c.trim());
        let t = '<table><thead><tr>' + cells(l).map((c) => '<th>' + inline(c, base) + '</th>').join('') + '</tr></thead><tbody>';
        i += 2;
        while (i < lines.length && /^\s*\|.*\|\s*$/.test(lines[i])) t += '<tr>' + cells(lines[i++]).map((c) => '<td>' + inline(c, base) + '</td>').join('') + '</tr>';
        out.push(t + '</tbody></table>'); continue;
      }
      if ((m = l.match(/^(\s*)([-*+]|\d+[.)])\s+(.*)$/))) {
        flush(); const ordered = /\d/.test(m[2]); const items = [];
        while (i < lines.length && (m = lines[i].match(/^(\s*)([-*+]|\d+[.)])\s+(.*)$/))) {
          let text = m[3]; i++;
          while (i < lines.length && /^\s{2,}\S/.test(lines[i]) && !/^\s*([-*+]|\d+[.)])\s+/.test(lines[i])) text += ' ' + lines[i++].trim();
          const r = text.match(reqID);
          items.push('<li' + (r ? ' id="' + r[1] + '" class="req"' : '') + '>' + inline(text, base) + '</li>');
        }
        out.push((ordered ? '<ol>' : '<ul>') + items.join('') + (ordered ? '</ol>' : '</ul>')); continue;
      }
      if (/^\s*$/.test(l)) { flush(); i++; continue; }
      para.push(l.trim()); i++;
    }
    flush();
    return out.join('\n');
  }

  class GbiMd extends HTMLElement {
    connectedCallback() {
      const src = this.getAttribute('src');
      if (!src) return;
      const url = new URL(src, document.baseURI);
      fetch(url).then((r) => { if (!r.ok) throw new Error(r.status + ' ' + r.statusText); return r.text(); }).then((md) => {
        this.innerHTML = render(md, url.href);
        if (location.hash) { const el = document.getElementById(decodeURIComponent(location.hash.slice(1))); if (el) el.scrollIntoView(); }
        this.dispatchEvent(new CustomEvent('gbi-md-rendered', { bubbles: true }));
      }).catch((e) => { this.insertAdjacentHTML('beforeend', '<p style="color:#d1453b">Could not load ' + esc(src) + ': ' + esc(String(e.message || e)) + '</p>'); });
    }
  }
  if (!customElements.get('gbi-md')) customElements.define('gbi-md', GbiMd);
  window.gbiMarkdown = render;
  if (!document.getElementById('gbi-md-style')) {
    const st = document.createElement('style');
    st.id = 'gbi-md-style';
    st.textContent = 'gbi-md{display:block}gbi-md .req:target,gbi-md :target{background:#fff8e6;outline:2px solid #f1d9a3;border-radius:3px}gbi-md table{border-collapse:collapse}gbi-md td,gbi-md th{border:1px solid #e1e5e8;padding:4px 8px}gbi-md pre{background:#f4f6f7;padding:10px;border-radius:6px;overflow:auto}gbi-md blockquote{border-left:3px solid #e1e5e8;margin:0;padding-left:12px;color:#55606a}';
    document.head.appendChild(st);
  }
})();
