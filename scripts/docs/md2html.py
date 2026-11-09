import html, re, sys

src, dst = sys.argv[1], sys.argv[2]
lines = open(src, encoding="utf-8").read().splitlines()

def slug(t):
    return re.sub(r"[^a-z0-9]+", "-", t.lower()).strip("-")

def inline(t):
    t = html.escape(t, quote=False)
    t = re.sub(r"`([^`]+)`", r"<code>\1</code>", t)
    t = re.sub(r"\*\*([^*]+)\*\*", r"<strong>\1</strong>", t)
    t = re.sub(r"(?<![\w*])\*([^*]+)\*(?![\w*])", r"<em>\1</em>", t)
    # highlight requirement IDs at the start of a list item / paragraph
    t = re.sub(r"<strong>((?:FR|NFR)-\d+[a-z]?)</strong>", r'<strong class="rid" id="\1">\1</strong>', t)
    t = re.sub(r"\b(SHALL NOT|SHALL|SHOULD|MAY)\b", r'<span class="kw kw-\1">\1</span>', t)
    t = t.replace('kw-SHALL NOT', 'kw-SHALLNOT')
    return t

out, toc = [], []
title = "Requirements"
i = 0
list_stack = []  # (type, indent)

def close_lists(to_indent=-1):
    while list_stack and list_stack[-1][1] > to_indent:
        typ, _ = list_stack.pop()
        out.append(f"</li></{typ}>")

def split_row(r):
    return [c.strip() for c in r.strip().strip("|").split("|")]

while i < len(lines):
    line = lines[i]
    m_li = re.match(r"^(\s*)([-*]|\d+\.)\s+(.*)$", line)
    if m_li:
        indent = len(m_li.group(1))
        typ = "ol" if m_li.group(2)[0].isdigit() else "ul"
        if not list_stack or indent > list_stack[-1][1]:
            out.append(f"<{typ}><li>")
            list_stack.append((typ, indent))
        else:
            close_lists(indent)
            if list_stack and list_stack[-1][1] == indent:
                out.append("</li><li>")
            else:
                out.append(f"<{typ}><li>")
                list_stack.append((typ, indent))
        out.append(inline(m_li.group(3)))
        i += 1
        continue
    close_lists()
    if not line.strip():
        i += 1
        continue
    m_h = re.match(r"^(#{1,6})\s+(.*)$", line)
    if m_h:
        lvl, text = len(m_h.group(1)), m_h.group(2)
        if lvl == 1:
            title = text
            out.append(f"<h1>{inline(text)}</h1>")
        else:
            sid = slug(text)
            if lvl == 2:
                toc.append((sid, text))
            out.append(f'<h{lvl} id="{sid}">{inline(text)}</h{lvl}>')
        i += 1
        continue
    if line.startswith("|") and i + 1 < len(lines) and re.match(r"^\|[\s:|-]+\|$", lines[i + 1]):
        head = split_row(line)
        i += 2
        rows = []
        while i < len(lines) and lines[i].startswith("|"):
            rows.append(split_row(lines[i]))
            i += 1
        out.append('<div class="table-wrap"><table><thead><tr>' +
                   "".join(f"<th>{inline(c)}</th>" for c in head) + "</tr></thead><tbody>")
        for r in rows:
            out.append("<tr>" + "".join(f"<td>{inline(c)}</td>" for c in r) + "</tr>")
        out.append("</tbody></table></div>")
        continue
    para = [line]
    i += 1
    while i < len(lines) and lines[i].strip() and not re.match(r"^(#|\||\s*[-*]\s|\s*\d+\.\s)", lines[i]):
        para.append(lines[i]); i += 1
    out.append(f"<p>{inline(' '.join(para))}</p>")
close_lists()

body = "\n".join(out)
# Put status line (first paragraph after h1) into a subtitle
body = body.replace("<p>Status:", '<p class="meta">Status:', 1)
nav = "\n".join(f'<li><a href="#{s}">{html.escape(t)}</a></li>' for s, t in toc)

page = f"""<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{html.escape(title)}</title>
<style>
:root {{
  --bg: #fbfaf7; --surface: #ffffff; --text: #1f2328; --muted: #656d76;
  --border: #e3e0d8; --accent: #2f6f5e; --accent-soft: #e6f0ec;
  --code-bg: #f2efe8; --shall: #9a3412; --should: #1d4ed8; --may: #6b7280;
}}
@media (prefers-color-scheme: dark) {{
  :root {{
    --bg: #15171a; --surface: #1c1f23; --text: #e6e6e3; --muted: #9aa1a9;
    --border: #2d3238; --accent: #6fc2a8; --accent-soft: #1f2e29;
    --code-bg: #262a30; --shall: #fb923c; --should: #7aa2ff; --may: #9ca3af;
  }}
}}
* {{ box-sizing: border-box; }}
html {{ scroll-behavior: smooth; }}
body {{ margin: 0; background: var(--bg); color: var(--text);
  font: 16px/1.6 -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; }}
.layout {{ display: grid; grid-template-columns: 260px minmax(0, 1fr); max-width: 1180px; margin: 0 auto; gap: 48px; padding: 0 24px; }}
nav {{ position: sticky; top: 0; align-self: start; max-height: 100vh; overflow-y: auto; padding: 40px 0; }}
nav h2 {{ font-size: 12px; text-transform: uppercase; letter-spacing: .08em; color: var(--muted); margin: 0 0 12px; border: 0; padding: 0; }}
nav ol {{ list-style: none; padding: 0; margin: 0; }}
nav a {{ display: block; padding: 5px 10px; border-radius: 6px; color: var(--muted); text-decoration: none; font-size: 14px; }}
nav a:hover {{ background: var(--accent-soft); color: var(--accent); }}
main {{ padding: 40px 0 96px; min-width: 0; }}
h1 {{ font-size: 2.2rem; line-height: 1.2; margin: 0 0 8px; letter-spacing: -.01em; }}
.meta {{ color: var(--muted); margin-top: 0; font-size: 14px; }}
h2 {{ font-size: 1.45rem; margin: 56px 0 16px; padding-bottom: 8px; border-bottom: 1px solid var(--border); scroll-margin-top: 16px; }}
h3 {{ font-size: 1.1rem; margin: 32px 0 10px; color: var(--accent); scroll-margin-top: 16px; }}
p, li {{ max-width: 78ch; }}
ul, ol {{ padding-left: 1.4em; }}
li {{ margin: 6px 0; }}
code {{ font: 0.88em ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; background: var(--code-bg); padding: 1px 5px; border-radius: 4px; }}
.rid {{ display: inline-block; font: 600 12px ui-monospace, SFMono-Regular, Menlo, monospace; color: var(--accent); background: var(--accent-soft); padding: 1px 7px; border-radius: 999px; margin-right: 2px; scroll-margin-top: 16px; }}
.kw {{ font-size: 0.78em; font-weight: 700; letter-spacing: .04em; }}
.kw-SHALL, .kw-SHALLNOT {{ color: var(--shall); }}
.kw-SHOULD {{ color: var(--should); }}
.kw-MAY {{ color: var(--may); }}
.table-wrap {{ overflow-x: auto; margin: 16px 0; border: 1px solid var(--border); border-radius: 8px; background: var(--surface); }}
table {{ border-collapse: collapse; width: 100%; font-size: 15px; }}
th, td {{ text-align: left; padding: 10px 14px; border-bottom: 1px solid var(--border); vertical-align: top; }}
th {{ font-size: 13px; text-transform: uppercase; letter-spacing: .05em; color: var(--muted); background: var(--code-bg); }}
tr:last-child td {{ border-bottom: 0; }}
td:first-child {{ white-space: nowrap; }}
@media (max-width: 860px) {{
  .layout {{ grid-template-columns: 1fr; gap: 0; padding: 0 16px; }}
  nav {{ position: static; max-height: none; padding: 24px 0 0; }}
  nav ol {{ display: flex; flex-wrap: wrap; gap: 4px; }}
  td:first-child {{ white-space: normal; }}
  h1 {{ font-size: 1.7rem; }}
}}
@media print {{
  nav {{ display: none; }}
  .layout {{ display: block; }}
  body {{ background: #fff; }}
}}
</style>
</head>
<body>
<div class="layout">
<nav aria-label="Contents"><h2>Contents</h2><ol>
{nav}
</ol></nav>
<main>
{body}
</main>
</div>
</body>
</html>
"""
open(dst, "w", encoding="utf-8").write(page)
print("wrote", dst)
