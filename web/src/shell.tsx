import { useEffect, useRef, useState, type ReactNode, createContext, useContext } from 'react';
import { api } from './api.gen';
import { useConnected, useData } from './live';
import { navigate, usePath } from './router';
import { A } from './ui';
import { GOTO, clickable, useDialog, useGlobalKeys, useShortcuts } from './keys';

export type SystemInfo = { version: string; actor: { name: string; kind: string; origin: string; login?: string }; owner: string; liveSessions: number; autoDispatch: boolean; experimental: any; dataDir: string; listen: string };
const SysCtx = createContext<SystemInfo | undefined>(undefined);
export function useSystem() { return useContext(SysCtx); }

export function SystemProvider({ children }: { children: ReactNode }) {
  const { data } = useData<SystemInfo>(() => api.system.info(), [], (e) => e.type.startsWith('session') || e.type.startsWith('settings') || e.type.startsWith('dispatcher'));
  return <SysCtx.Provider value={data}>{children}</SysCtx.Provider>;
}

function Identity() {
  const sys = useSystem();
  const live = useConnected();
  if (!sys) return <span className="who">…</span>;
  const a = sys.actor;
  const label = a.origin === 'tailscale' ? `Tailnet · ${a.login || a.name}` : a.origin === 'local' ? `Local · ${a.name}` : `${a.origin} · ${a.name}`;
  return <span className={'who ' + (live ? '' : 'off')} title={live ? 'Live updates connected (FR-36)' : 'Live updates disconnected: retrying'}>● {label}</span>;
}

const NAV: [string, string, string?][] = [['Projects', '/'], ['Questions', '/questions', 'q'], ['Sessions', '/sessions', 's'], ['Queue', '/queue'], ['Reviews', '/reviews', 'r'], ['Options', '/options', 'o'], ['Rehearsals', '/rehearsals', 'g']];
const ORG: [string, string][] = [['Archetypes', '/archetypes'], ['Standards', '/standards'], ['History', '/history']];
const SYS: [string, string][] = [['Settings', '/settings'], ['Docs', '/docs']];

export function Shell({ title, crumb, right, tabs, children, noPad }: { title: ReactNode; crumb?: ReactNode; right?: ReactNode; tabs?: ReactNode; children: ReactNode; noPad?: boolean }) {
  const path = usePath();
  const sys = useSystem();
  const [pal, setPal] = useState(false);
  const [help, setHelp] = useState(false);
  useGlobalKeys();
  useShortcuts({ search: () => setPal(true), help: () => setHelp(true), go: navigate });
  const { data: inbox } = useData<any[]>(() => api.inbox.list(), [], (e) => /^(question|review|rehearsal|options|value|task)\./.test(e.type));
  const count = (k: string) => (inbox || []).filter((i) => i.kind === ({ q: 'question', r: 'review', o: 'options', g: 'gate' } as any)[k]).length;
  const blocking = (inbox || []).filter((i) => i.blocking).length;
  useEffect(() => {
    const k = (e: KeyboardEvent) => { if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') { e.preventDefault(); setPal((p) => !p); } };
    window.addEventListener('keydown', k); return () => window.removeEventListener('keydown', k);
  }, []);
  const on = (to: string) => (to === '/' ? path === '/' || path.startsWith('/projects') : path.startsWith(to));
  const link = ([label, to, badge]: [string, string, string?]) => {
    let b: ReactNode = null;
    if (badge === 's' && sys?.liveSessions) b = <span className="badge n">{sys.liveSessions}</span>;
    else if (badge && badge !== 's') { const n = count(badge); if (n) b = <span className={'badge ' + (badge === 'q' && blocking ? '' : 'n')}>{n}</span>; }
    return <A key={to} to={to} className={on(to) ? 'on' : ''}>{label}{b}</A>;
  };
  return <div className="app">
    <a className="skip" href="#main" onClick={(e) => { e.preventDefault(); document.getElementById('main')?.focus(); }}>Skip to content</a>
    <aside className="side" aria-label="Main navigation">
      <div className="brand"><b>gb</b>go-build-it</div>
      {NAV.map(link)}
      <div className="sep">Organization</div>{ORG.map(link)}
      <div className="sep">System</div>{SYS.map(link)}
      <div className="foot">{sys ? <>v{sys.version}<br />{sys.liveSessions} live · dispatch {sys.autoDispatch ? 'auto' : 'manual'}<br /><a href="/m/" style={{ color: 'inherit' }}>Mobile UI ↗</a></> : '…'}</div>
    </aside>
    <div className="main" id="main" tabIndex={-1}>
      <div className="topbar"><h1>{crumb && <span className="crumb">{crumb} / </span>}{title}</h1>{right}
        <button type="button" className="search" onClick={() => setPal(true)} aria-keyshortcuts="Meta+K /">Search, run a command…  ⌘K</button>
        <button type="button" className="lnk small" onClick={() => setHelp(true)} title="Keyboard shortcuts (?)" aria-keyshortcuts="?">Keys <kbd>?</kbd></button><Identity /></div>
      {tabs}
      {noPad ? children : <div className="content">{children}</div>}
    </div>
    {pal && <Palette onClose={() => setPal(false)} />}
    {help && <KeysHelp onClose={() => setHelp(false)} />}
  </div>;
}

export function ProjectTabs({ id, on }: { id: string; on: string }) {
  const t: [string, string][] = [['overview', 'Overview'], ['artifacts', 'Artifacts'], ['team', 'Team'], ['queue', 'Queue'], ['sessions', 'Sessions'], ['releases', 'Releases'], ['history', 'History'], ['settings', 'Settings']];
  return <nav className="tabs">{t.map(([k, l]) => <A key={k} to={k === 'overview' ? `/projects/${id}` : `/projects/${id}/${k}`} className={on === k ? 'on' : ''}>{l}</A>)}</nav>;
}

// Keyboard shortcuts overview (press ?). Matches docs/user/18-keyboard.md.
function KeysHelp({ onClose }: { onClose: () => void }) {
  const dlg = useRef<HTMLDivElement>(null);
  useDialog(dlg, onClose);
  const row = (k: ReactNode, what: string) => <tr><td>{k}</td><td>{what}</td></tr>;
  return <div className="overlay" onClick={(e) => { if (e.target === e.currentTarget) onClose(); }}>
    <div className="keys-help" role="dialog" aria-modal="true" aria-label="Keyboard shortcuts" ref={dlg}>
      <div className="row c"><h2>Keyboard shortcuts</h2><span className="right" /><button type="button" className="btn" onClick={onClose}>✕ Esc</button></div>
      <h3>Anywhere</h3><table><tbody>
        {row(<><kbd>⌘</kbd> <kbd>K</kbd> or <kbd>/</kbd></>, 'Search and run commands')}
        {row(<kbd>?</kbd>, 'Show this list')}
        {row(<><kbd>Tab</kbd> / <kbd>Shift</kbd> <kbd>Tab</kbd></>, 'Move between controls; the first Tab offers "Skip to content"')}
        {GOTO.map(([k, , label]) => <tr key={k}><td><kbd>g</kbd> then <kbd>{k}</kbd></td><td>Go to {label}</td></tr>)}
      </tbody></table>
      <h3>Lists, tables and cards</h3><table><tbody>
        {row(<><kbd>↑</kbd> <kbd>↓</kbd> or <kbd>j</kbd> <kbd>k</kbd></>, 'Previous or next item')}
        {row(<><kbd>←</kbd> <kbd>→</kbd></>, 'Previous or next card in the project grid')}
        {row(<><kbd>Home</kbd> <kbd>End</kbd></>, 'First or last item')}
        {row(<kbd>Enter</kbd>, 'Open the focused item')}
      </tbody></table>
      <h3>Tabs and toggles</h3><table><tbody>
        {row(<><kbd>←</kbd> <kbd>→</kbd></>, 'Move between tabs or options (segmented options switch as you move)')}
        {row(<><kbd>Enter</kbd> or <kbd>Space</kbd></>, 'Activate the focused button')}
      </tbody></table>
      <h3>Text boxes</h3><table><tbody>
        {row(<><kbd>⌘</kbd> <kbd>Enter</kbd></>, 'Send, answer or save (the panel\'s main button)')}
        {row(<kbd>Esc</kbd>, 'Close the dialog you are in')}
      </tbody></table>
      <h3>Viewer</h3><table><tbody>
        {row(<><kbd>←</kbd> <kbd>→</kbd></>, 'Previous or next attachment')}
        {row(<><kbd>↑</kbd> <kbd>↓</kbd></>, 'Switch attachments while the thumbnail strip has focus')}
        {row(<kbd>Esc</kbd>, 'Close the viewer')}
      </tbody></table>
      <p className="small muted">Single-key shortcuts are ignored while you type in a text field.</p>
    </div>
  </div>;
}

// Command palette: search everything; every hit shows its CLI equivalent (FR-30; borrowed from Approach C).
function Palette({ onClose }: { onClose: () => void }) {
  const [q, setQ] = useState('');
  const [hits, setHits] = useState<any[]>([]);
  const [sel, setSel] = useState(0);
  useEffect(() => {
    const t = setTimeout(() => { if (q.trim()) api.search.all({ q }).then(setHits).catch(() => setHits([])); else setHits([]); }, 120);
    return () => clearTimeout(t);
  }, [q]);
  const go = (h: any) => { if (h.link) { navigate(h.link); onClose(); } else { navigator.clipboard?.writeText(h.cli); } };
  const dlg = useRef<HTMLDivElement>(null);
  useDialog(dlg, onClose);
  return <div className="overlay" onClick={(e) => { if (e.target === e.currentTarget) onClose(); }}>
    <div className="palette" role="dialog" aria-modal="true" aria-label="Command palette" ref={dlg}>
      <input autoFocus placeholder="Search projects, tasks, docs, commands…" value={q} onChange={(e) => { setQ(e.target.value); setSel(0); }}
        onKeyDown={(e) => { if (e.key === 'Escape') onClose(); if (e.key === 'ArrowDown') setSel((s) => Math.min(hits.length - 1, s + 1)); if (e.key === 'ArrowUp') setSel((s) => Math.max(0, s - 1)); if (e.key === 'Enter' && hits[sel]) go(hits[sel]); }} />
      <div className="res">
        {hits.map((h, k) => <div key={k} className={'hit ' + (k === sel ? 'on' : '')} {...clickable(() => go(h))}>
          <span className="chip">{h.kind}</span><span className="grow">{h.title}</span><code className="muted">{h.cli}</code></div>)}
        {q && hits.length === 0 && <div className="empty">No matches</div>}
        {!q && <div className="empty small">Type to search. Commands show the matching <code>gbi</code> CLI call; Enter copies it.</div>}
      </div>
    </div>
  </div>;
}
