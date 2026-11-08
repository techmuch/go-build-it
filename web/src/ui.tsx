import { useEffect, useMemo, useRef, useState, type ReactNode, createContext, useContext } from 'react';
import { marked } from 'marked';
import { api } from './api.gen';
import { navigate } from './router';

// ---------- formatting ----------
export function ago(ts?: string): string {
  if (!ts) return '';
  const s = (Date.now() - new Date(ts).getTime()) / 1000;
  if (s < 60) return 'just now';
  if (s < 3600) return `${Math.floor(s / 60)}m ago`;
  if (s < 86400) return `${Math.floor(s / 3600)}h ago`;
  return `${Math.floor(s / 86400)}d ago`;
}
export function clock(ts?: string) { return ts ? new Date(ts).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) : ''; }
export function day(ts?: string) { return ts ? new Date(ts).toLocaleDateString([], { month: 'short', day: 'numeric' }) : ''; }
export function initials(name: string) { return (name || '?').replace(/[^A-Za-z ]/g, '').slice(0, 2).replace(/^./, (c) => c.toUpperCase()) || '?'; }
export function kfmt(n: number) { return n >= 1000000 ? (n / 1000000).toFixed(1) + 'M' : n >= 1000 ? Math.round(n / 1000) + 'k' : String(n); }

// ---------- small components ----------
export function Chip({ c, children, title }: { c?: string; children: ReactNode; title?: string }) { return <span className={'chip ' + (c || '')} title={title}>{children}</span>; }
export function Bar({ pct, warn }: { pct: number; warn?: number }) {
  const cls = pct >= 90 ? 'bar hot' : warn !== undefined && pct >= warn ? 'bar warn' : 'bar';
  return <div className={cls}><i style={{ width: Math.min(100, Math.max(0, pct)) + '%' }} /></div>;
}
export function Av({ name, role, s }: { name: string; role?: string; s?: boolean }) { return <span className={`av ${role || ''} ${s ? 's' : ''}`} title={name}>{initials(name)}</span>; }
export function Card({ title, right, children, style, className }: { title?: ReactNode; right?: ReactNode; children: ReactNode; style?: React.CSSProperties; className?: string }) {
  return <div className={'card ' + (className || '')} style={style}>{(title || right) && <h3>{title}<span style={{ textTransform: 'none', letterSpacing: 0 }}>{right}</span></h3>}{children}</div>;
}
export function Seg<T extends string>({ value, options, onChange }: { value: T; options: [T, string][]; onChange: (v: T) => void }) {
  return <span className="seg">{options.map(([v, l]) => <button key={v} className={v === value ? 'on' : ''} onClick={() => onChange(v)} type="button">{l}</button>)}</span>;
}
export function Toggle({ on, onChange, label }: { on: boolean; onChange: (v: boolean) => void; label?: string }) {
  return <button type="button" aria-label={label} aria-pressed={on} className={'toggle ' + (on ? '' : 'off')} onClick={() => onChange(!on)} />;
}
export function Empty({ children }: { children: ReactNode }) { return <div className="empty">{children}</div>; }
export function Err({ msg }: { msg?: string }) { return msg ? <div className="err">{msg}</div> : null; }
export function Loading({ what }: { what?: string }) { return <div className="empty">Loading{what ? ' ' + what : ''}…</div>; }

const STATUS: Record<string, string> = {
  queued: '', running: 'acc', blocked: 'red', review: 'amb', done: 'grn', failed: 'red', paused: 'amb',
  open: 'amb', answered: 'grn', skipped: '', pending: 'amb', approved: 'grn', changes: 'red', rejected: 'red', decided: 'grn',
  rate_limited: 'amb', cancelled: '', active: 'acc', planned: '', starting: 'acc', met: 'grn', partly: 'amb', not: 'red', unassessed: '',
  accepted: 'grn', proposed: 'acc', dismissed: '',
};
export function Status({ s }: { s: string }) { return <Chip c={STATUS[s] ?? ''}>{s.replace('_', ' ')}</Chip>; }

export function A({ to, children, className }: { to: string; children: ReactNode; className?: string }) {
  return <a href={to} className={className} onClick={(e) => { if (e.metaKey || e.ctrlKey) return; e.preventDefault(); navigate(to); }}>{children}</a>;
}

// ---------- toasts ----------
const ToastCtx = createContext<(msg: string, err?: boolean) => void>(() => {});
export function ToastProvider({ children }: { children: ReactNode }) {
  const [t, setT] = useState<{ msg: string; err?: boolean } | null>(null);
  useEffect(() => { if (t) { const id = setTimeout(() => setT(null), 4000); return () => clearTimeout(id); } }, [t]);
  return <ToastCtx.Provider value={(msg, err) => setT({ msg, err })}>{children}{t && <div className={'toast ' + (t.err ? 'err' : '')} role="status">{t.msg}</div>}</ToastCtx.Provider>;
}
export function useToast() { return useContext(ToastCtx); }
export function useAction() {
  const toast = useToast();
  const [busy, setBusy] = useState(false);
  const run = async <T,>(fn: () => Promise<T>, ok?: string): Promise<T | undefined> => {
    setBusy(true);
    try { const r = await fn(); if (ok) toast(ok); return r; } catch (e: any) { toast(e.message || String(e), true); return undefined; } finally { setBusy(false); }
  };
  return { busy, run };
}

// ---------- dictation (FR-107) ----------
type SR = { start(): void; stop(): void; onresult: (e: any) => void; onend: () => void; onerror: (e: any) => void; continuous: boolean; interimResults: boolean; lang: string };
export function speechSupported() { return !!((window as any).SpeechRecognition || (window as any).webkitSpeechRecognition); }
export function Mic({ onText }: { onText: (t: string) => void }) {
  const [rec, setRec] = useState(false);
  const r = useRef<SR | null>(null);
  const toast = useToast();
  const toggle = () => {
    if (rec) { r.current?.stop(); return; }
    const C = (window as any).SpeechRecognition || (window as any).webkitSpeechRecognition;
    if (!C) { toast('Dictation needs a browser with speech recognition (Chrome, Edge or Safari).', true); return; }
    const s: SR = new C();
    s.continuous = true; s.interimResults = false; s.lang = navigator.language || 'en-US';
    s.onresult = (e: any) => { let t = ''; for (let i = e.resultIndex; i < e.results.length; i++) if (e.results[i].isFinal) t += e.results[i][0].transcript; if (t) onText(t.trim()); };
    s.onend = () => setRec(false);
    s.onerror = (e: any) => { setRec(false); if (e.error !== 'aborted') toast('Dictation: ' + e.error, true); };
    r.current = s; s.start(); setRec(true);
  };
  return <button type="button" className={'mic ' + (rec ? 'rec' : '')} onClick={toggle} title={rec ? 'Stop dictation' : 'Dictate (speech to text)'} aria-label="Dictate">🎙</button>;
}
export function DictArea({ value, onChange, placeholder, rows, autoFocus }: { value: string; onChange: (v: string) => void; placeholder?: string; rows?: number; autoFocus?: boolean }) {
  return <div className="dict"><textarea value={value} rows={rows || 3} autoFocus={autoFocus} placeholder={placeholder} onChange={(e) => onChange(e.target.value)} /><Mic onText={(t) => onChange(value ? value.replace(/\s*$/, ' ') + t : t)} /></div>;
}

// ---------- markdown ----------
export function Markdown({ src, className }: { src: string; className?: string }) {
  const html = useMemo(() => marked.parse(src || '', { async: false }) as string, [src]);
  return <div className={'md ' + (className || '')} dangerouslySetInnerHTML={{ __html: html }} />;
}

// ---------- viewer (FR-102a, FR-116a) ----------
export type Attachment = { name: string; kind: string; url?: string; path?: string; body?: string };
type Pin = { x: number; y: number; text: string; who: string };
const SIZES: Record<string, number> = { desktop: 1280, tablet: 820, phone: 390 };

export function kindOf(path: string): string {
  const e = path.split('.').pop()?.toLowerCase() || '';
  if (['html', 'htm'].includes(e)) return 'html';
  if (['png', 'jpg', 'jpeg', 'gif', 'svg', 'webp'].includes(e)) return 'image';
  if (e === 'json') return 'json';
  if (['diff', 'patch'].includes(e)) return 'diff';
  if (['log', 'txt'].includes(e)) return 'log';
  if (e === 'md') return 'markdown';
  return 'text';
}

export function Viewer({ items, index, onClose, title }: { items: Attachment[]; index: number; onClose: () => void; title?: string }) {
  const [i, setI] = useState(index);
  const [size, setSize] = useState<'desktop' | 'tablet' | 'phone'>('desktop');
  const [fit, setFit] = useState(true);
  const [pinMode, setPinMode] = useState(false);
  const [pins, setPins] = useState<Record<number, Pin[]>>({});
  const [draft, setDraft] = useState('');
  const [text, setText] = useState<string>('');
  const stage = useRef<HTMLDivElement>(null);
  const [stageW, setStageW] = useState(1000);
  const it = items[i];
  useEffect(() => {
    const k = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose(); if (e.key === 'ArrowRight') setI((x) => Math.min(items.length - 1, x + 1)); if (e.key === 'ArrowLeft') setI((x) => Math.max(0, x - 1)); };
    window.addEventListener('keydown', k); return () => window.removeEventListener('keydown', k);
  }, [items.length, onClose]);
  useEffect(() => {
    setText('');
    if (!it) return;
    if (it.body !== undefined) { setText(it.body); return; }
    if (it.url && ['json', 'diff', 'log', 'text', 'markdown'].includes(it.kind)) fetch(it.url).then((r) => r.text()).then(setText).catch(() => setText('(could not load)'));
  }, [it]);
  useEffect(() => {
    const ro = new ResizeObserver(() => setStageW((stage.current?.clientWidth || 1000) - 32));
    if (stage.current) ro.observe(stage.current);
    return () => ro.disconnect();
  }, []);
  if (!it) return null;
  const w = SIZES[size];
  const scale = fit && it.kind === 'html' ? Math.min(1, stageW / w) : 1;
  const myPins = pins[i] || [];
  const addPin = (e: React.MouseEvent) => {
    const r = (e.currentTarget as HTMLElement).getBoundingClientRect();
    const x = ((e.clientX - r.left) / r.width) * 100, y = ((e.clientY - r.top) / r.height) * 100;
    setPins({ ...pins, [i]: [...myPins, { x, y, text: draft || 'Comment', who: 'you' }] });
    setDraft('');
  };
  let body: ReactNode;
  if (it.kind === 'html') {
    body = <div style={{ position: 'relative', width: w * scale, height: 820 * scale }}>
      <iframe title={it.name} src={it.url} sandbox="allow-scripts allow-forms" style={{ width: w, height: 820, transform: `scale(${scale})`, transformOrigin: '0 0' }} />
      {pinMode && <div className="pinlayer" onClick={addPin} />}
      {myPins.map((p, k) => <div key={k} className="pin" style={{ left: `calc(${p.x}% - 11px)`, top: `calc(${p.y}% - 22px)` }}><span>{k + 1}</span></div>)}
    </div>;
  } else if (it.kind === 'image') {
    body = <div style={{ position: 'relative' }}><img src={it.url} alt={it.name} style={{ maxWidth: fit ? '100%' : 'none', display: 'block' }} />
      {pinMode && <div className="pinlayer" onClick={addPin} />}
      {myPins.map((p, k) => <div key={k} className="pin" style={{ left: `calc(${p.x}% - 11px)`, top: `calc(${p.y}% - 22px)` }}><span>{k + 1}</span></div>)}</div>;
  } else if (it.kind === 'diff') {
    body = <div className="card" style={{ width: '100%', padding: 0 }}><DiffView text={text} /></div>;
  } else if (it.kind === 'markdown') {
    body = <div className="card" style={{ width: '100%', maxWidth: 900 }}><Markdown src={text} /></div>;
  } else if (it.kind === 'link') {
    body = <div className="card"><a href={it.url} target="_blank" rel="noreferrer">{it.url}</a></div>;
  } else {
    let pretty = text;
    if (it.kind === 'json') { try { pretty = JSON.stringify(JSON.parse(text), null, 2); } catch { /* keep */ } }
    body = <pre className="term" style={{ width: '100%', borderRadius: 6, margin: 0 }}>{pretty}</pre>;
  }
  return <div className="overlay" role="dialog" aria-label="Viewer" onClick={(e) => { if (e.target === e.currentTarget) onClose(); }}>
    <div className="viewer">
      <div className="viewer-h">
        <b style={{ whiteSpace: 'nowrap' }}>{it.name}</b><span className="small muted" style={{ whiteSpace: 'nowrap' }}>{title ? title + ' · ' : ''}{i + 1} of {items.length}</span>
        {it.kind === 'html' && <Seg value={size} onChange={setSize} options={[['desktop', 'Desktop 1280'], ['tablet', 'Tablet 820'], ['phone', 'Phone 390']]} />}
        {(it.kind === 'html' || it.kind === 'image') && <><Seg value={fit ? 'fit' : '100'} onChange={(v) => setFit(v === 'fit')} options={[['fit', 'Fit'], ['100', '100%']]} />
          <button type="button" className={'btn sm ' + (pinMode ? 'p' : '')} onClick={() => setPinMode(!pinMode)}>📍 Pin</button></>}
        <span className="right" />
        {it.url && <a className="small" href={it.url} target="_blank" rel="noreferrer" style={{ whiteSpace: 'nowrap' }}>New tab ↗</a>}
        <button type="button" className="btn" onClick={onClose}>✕ Esc</button>
      </div>
      <div className="viewer-b">
        <div className="viewer-strip">{items.map((x, k) => <button type="button" key={k} className={k === i ? 'on' : ''} onClick={() => setI(k)}><div className="muted xs">{x.kind.toUpperCase()}</div>{x.name}</button>)}</div>
        <div className="viewer-stage" ref={stage}>{body}</div>
        <div className="viewer-side">
          <b className="small">Comments on this item</b>
          {myPins.length === 0 && <span className="muted small">Turn on 📍 Pin and click the item to place a comment.</span>}
          {myPins.map((p, k) => <div key={k} className="msg small"><span className="chip amb">{k + 1}</span><div className="bub"><b>{p.who}</b>: {p.text}</div></div>)}
          <DictArea value={draft} onChange={setDraft} placeholder="Comment for the next pin…" rows={2} />
          <span className="muted xs">Viewers: HTML (live, sandboxed), images (zoom), diffs, JSON and logs (highlighted), Markdown.</span>
        </div>
      </div>
    </div>
  </div>;
}

export function DiffView({ text }: { text: string }) {
  return <pre className="mono" style={{ margin: 0, padding: 10, overflow: 'auto', whiteSpace: 'pre' }}>{text.split('\n').map((l, k) =>
    <div key={k} style={{ background: l.startsWith('+') ? 'var(--ins)' : l.startsWith('-') ? 'var(--del)' : l.startsWith('@@') ? 'var(--chip)' : undefined }}>{l || ' '}</div>)}</pre>;
}

// ---------- rich editor (FR-45a) ----------
export function Editor({ value, onChange, mode, onMode, toolbarRight }: { value: string; onChange: (v: string) => void; mode: 'wysiwyg' | 'markdown' | 'split'; onMode: (m: 'wysiwyg' | 'markdown' | 'split') => void; toolbarRight?: ReactNode }) {
  const ta = useRef<HTMLTextAreaElement>(null);
  const wys = useRef<HTMLDivElement>(null);
  const lines = value.split('\n').length;
  // WYSIWYG mode edits rendered HTML; on blur it is converted back to Markdown.
  const html = useMemo(() => marked.parse(value || '', { async: false }) as string, [value]);
  useEffect(() => { if (mode === 'wysiwyg' && wys.current && document.activeElement !== wys.current) wys.current.innerHTML = html; }, [html, mode]);
  const wrap = (before: string, after = before, linePrefix = false) => {
    if (mode === 'wysiwyg') {
      const cmd: Record<string, [string, string?]> = { '**': ['bold'], '_': ['italic'], '# ': ['formatBlock', 'H1'], '## ': ['formatBlock', 'H2'], '- ': ['insertUnorderedList'], '1. ': ['insertOrderedList'] };
      const c = cmd[before];
      if (c) document.execCommand(c[0], false, c[1]);
      return;
    }
    const el = ta.current; if (!el) return;
    const s = el.selectionStart, e = el.selectionEnd;
    let next: string;
    if (linePrefix) { const ls = value.lastIndexOf('\n', s - 1) + 1; next = value.slice(0, ls) + before + value.slice(ls); }
    else next = value.slice(0, s) + before + value.slice(s, e) + after + value.slice(e);
    onChange(next);
    setTimeout(() => { el.focus(); el.setSelectionRange(s + before.length, e + before.length); }, 0);
  };
  const tb = (label: ReactNode, fn: () => void, title: string) => <button type="button" className="tb" title={title} onMouseDown={(e) => e.preventDefault()} onClick={fn}>{label}</button>;
  const md = <div className="code"><div className="ln">{Array.from({ length: lines }, (_, k) => k + 1).join('\n')}</div>
    <textarea ref={ta} spellCheck value={value} onChange={(e) => onChange(e.target.value)} aria-label="Markdown source" /></div>;
  return <div>
    <div className="edtool">
      <Seg value={mode} onChange={onMode} options={[['wysiwyg', 'WYSIWYG'], ['markdown', 'Markdown'], ['split', 'Split']]} />
      <span className="s" />
      {tb('H1', () => wrap('# ', '', true), 'Heading 1')}{tb('H2', () => wrap('## ', '', true), 'Heading 2')}
      {tb('B', () => wrap('**'), 'Bold')}{tb(<i>I</i>, () => wrap('_'), 'Italic')}
      {tb('• List', () => wrap('- ', '', true), 'Bullet list')}{tb('1. List', () => wrap('1. ', '', true), 'Numbered list')}
      {tb('⊞ Table', () => wrap('\n| Column | Column |\n|---|---|\n| | |\n', ''), 'Table')}{tb('🔗', () => wrap('[', '](https://)'), 'Link')}
      {tb('FR#', () => wrap('**FR-', '**: '), 'Requirement ID')}
      <span className="right small">{toolbarRight}</span>
    </div>
    {mode === 'markdown' && md}
    {mode === 'wysiwyg' && <div ref={wys} className="wys md" contentEditable suppressContentEditableWarning aria-label="Document editor"
      onBlur={(e) => onChange(htmlToMarkdown(e.currentTarget))} />}
    {mode === 'split' && <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 0 }}>{md}<div className="wys" style={{ borderLeft: 0 }}><Markdown src={value} /></div></div>}
  </div>;
}

// htmlToMarkdown converts the WYSIWYG editor's DOM back to Markdown (covers the toolbar's elements).
export function htmlToMarkdown(root: HTMLElement): string {
  const walk = (n: Node): string => {
    if (n.nodeType === Node.TEXT_NODE) return (n.textContent || '').replace(/ /g, ' ');
    if (!(n instanceof HTMLElement)) return '';
    const kids = () => Array.from(n.childNodes).map(walk).join('');
    switch (n.tagName) {
      case 'H1': return `# ${kids().trim()}\n\n`;
      case 'H2': return `## ${kids().trim()}\n\n`;
      case 'H3': return `### ${kids().trim()}\n\n`;
      case 'H4': return `#### ${kids().trim()}\n\n`;
      case 'P': case 'DIV': return `${kids().trim()}\n\n`;
      case 'BR': return '\n';
      case 'STRONG': case 'B': return `**${kids()}**`;
      case 'EM': case 'I': return `_${kids()}_`;
      case 'CODE': return n.parentElement?.tagName === 'PRE' ? kids() : '`' + kids() + '`';
      case 'PRE': return '```\n' + kids().trimEnd() + '\n```\n\n';
      case 'A': return `[${kids()}](${n.getAttribute('href') || ''})`;
      case 'UL': return Array.from(n.children).map((li) => `- ${walk(li).trim()}`).join('\n') + '\n\n';
      case 'OL': return Array.from(n.children).map((li, k) => `${k + 1}. ${walk(li).trim()}`).join('\n') + '\n\n';
      case 'LI': return kids();
      case 'BLOCKQUOTE': return kids().trim().split('\n').map((l) => '> ' + l).join('\n') + '\n\n';
      case 'HR': return '---\n\n';
      case 'TABLE': {
        const rows = Array.from(n.querySelectorAll('tr')).map((tr) => '| ' + Array.from(tr.children).map((c) => walk(c).trim()).join(' | ') + ' |');
        if (rows.length > 0) rows.splice(1, 0, '|' + rows[0].split('|').slice(1, -1).map(() => '---').join('|') + '|');
        return rows.join('\n') + '\n\n';
      }
      case 'IMG': return `![${n.getAttribute('alt') || ''}](${n.getAttribute('src') || ''})`;
      default: return kids();
    }
  };
  return Array.from(root.childNodes).map(walk).join('').replace(/\n{3,}/g, '\n\n').trim() + '\n';
}

// ---------- agent chat with proposals (FR-45a, FR-13, FR-51) ----------
export type Hunk = { start: number; removed: string[]; added: string[]; context: string };
export type ChatMsg = { who: string; me: boolean; text: string };
export function AgentChat({ title, personas, persona, setPersona, send, suggestions, onProposal, placeholder }: {
  title: string; personas: { id: string; name: string; role: string }[]; persona: string; setPersona: (p: string) => void;
  send: (msg: string, history: string[]) => Promise<{ message: string; document?: string; hunks?: Hunk[]; persona: string }>;
  suggestions?: string[]; onProposal: (doc: string, hunks: Hunk[]) => void; placeholder?: string;
}) {
  const [msgs, setMsgs] = useState<ChatMsg[]>([]);
  const [text, setText] = useState('');
  const [busy, setBusy] = useState(false);
  const toast = useToast();
  const end = useRef<HTMLDivElement>(null);
  useEffect(() => { end.current?.scrollIntoView({ block: 'nearest' }); }, [msgs, busy]);
  const go = async (m: string) => {
    if (!m.trim() || busy) return;
    const hist = msgs.map((x) => `${x.me ? 'Human' : x.who}: ${x.text}`);
    setMsgs((x) => [...x, { who: 'you', me: true, text: m }]); setText(''); setBusy(true);
    try {
      const r = await send(m, hist);
      setMsgs((x) => [...x, { who: r.persona, me: false, text: r.message + (r.hunks?.length ? `\n\n(${r.hunks.length} proposed change${r.hunks.length > 1 ? 's' : ''} highlighted in the document)` : '') }]);
      if (r.document !== undefined && r.hunks?.length) onProposal(r.document, r.hunks);
    } catch (e: any) { toast(e.message, true); setMsgs((x) => [...x, { who: 'go-build-it', me: false, text: 'Could not reach the agent: ' + e.message }]); }
    finally { setBusy(false); }
  };
  const p = personas.find((x) => x.id === persona);
  return <div className="card" style={{ display: 'flex', flexDirection: 'column', minHeight: 420 }}>
    <h3>{title}<select value={persona} onChange={(e) => setPersona(e.target.value)} aria-label="Persona" style={{ padding: '2px 6px', fontSize: 12 }}>
      {personas.map((x) => <option key={x.id} value={x.id}>{x.name} · {x.role}</option>)}</select></h3>
    <div className="chat" style={{ flex: 1, maxHeight: 520 }}>
      {msgs.length === 0 && <div className="muted small">Ask {p?.name || 'a persona'} for a change. Proposals appear highlighted in the document for you to accept or reject. You can also edit directly.</div>}
      {msgs.map((m, k) => <div key={k} className={'msg ' + (m.me ? 'me' : '')}><Av s name={m.who} role={m.me ? 'human' : p?.role} /><div className="bub">{m.text}</div></div>)}
      {busy && <div className="msg"><Av s name={p?.name || '?'} role={p?.role} /><div className="bub muted">thinking…</div></div>}
      <div ref={end} />
    </div>
    {suggestions && <div className="small muted" style={{ margin: '8px 0 4px', display: 'flex', gap: 4, flexWrap: 'wrap' }}>Try: {suggestions.map((s) => <button type="button" key={s} className="chip" style={{ border: 0, cursor: 'pointer' }} onClick={() => go(s)}>{s}</button>)}</div>}
    <div className="compose"><textarea value={text} placeholder={placeholder || 'Ask for a change…'} onChange={(e) => setText(e.target.value)}
      onKeyDown={(e) => { if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) go(text); }} aria-label="Message" />
      <Mic onText={(t) => setText((v) => (v ? v + ' ' : '') + t)} /><button type="button" className="btn p" disabled={busy} onClick={() => go(text)}>Send</button></div>
  </div>;
}

export function Proposals({ hunks, accepted, setAccepted }: { hunks: Hunk[]; accepted: Set<number>; setAccepted: (s: Set<number>) => void }) {
  if (!hunks.length) return null;
  const toggle = (k: number, on: boolean) => { const s = new Set(accepted); if (on) s.add(k); else s.delete(k); setAccepted(s); };
  return <div className="card" style={{ borderColor: 'var(--acc)' }}>
    <h3>Proposed changes · {hunks.length}<span><button type="button" className="btn sm" onClick={() => setAccepted(new Set(hunks.map((_, k) => k)))}>Accept all</button> <button type="button" className="btn sm" onClick={() => setAccepted(new Set())}>Reject all</button></span></h3>
    {hunks.map((h, k) => <div className="hunk" key={k}>
      {h.context && <div className="muted" style={{ padding: '0 8px' }}>… {h.context}</div>}
      {h.removed.map((l, j) => <div key={'r' + j} className="del">{l || ' '}</div>)}
      {h.added.map((l, j) => <div key={'a' + j} className="ins">{l || ' '}</div>)}
      <div className="bar2"><span className="small muted">change {k + 1}</span><span className="right" />
        <button type="button" className={'btn sm ' + (accepted.has(k) ? 'p' : '')} onClick={() => toggle(k, true)}>✓ Accept</button>
        <button type="button" className={'btn sm ' + (!accepted.has(k) ? 'd' : '')} onClick={() => toggle(k, false)}>✕ Reject</button></div>
    </div>)}
  </div>;
}

export async function applyHunks(original: string, hunks: Hunk[], accepted: Set<number>): Promise<string> {
  const r = await api.diff.apply({ original, hunks, accepted: [...accepted] });
  return r.content;
}
