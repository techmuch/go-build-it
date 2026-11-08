import { useEffect, useState, type ReactNode, createContext, useContext } from 'react';
import { api } from './api.gen';
import { useConnected, useData } from './live';
import { navigate, usePath } from './router';
import { A } from './ui';

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
const ORG: [string, string][] = [['Team', '/team'], ['Standards', '/standards'], ['History', '/history']];
const SYS: [string, string][] = [['Settings', '/settings'], ['Docs', '/docs']];

export function Shell({ title, crumb, right, tabs, children, noPad }: { title: ReactNode; crumb?: ReactNode; right?: ReactNode; tabs?: ReactNode; children: ReactNode; noPad?: boolean }) {
  const path = usePath();
  const sys = useSystem();
  const [pal, setPal] = useState(false);
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
    <aside className="side" aria-label="Main navigation">
      <div className="brand"><b>gb</b>go-build-it</div>
      {NAV.map(link)}
      <div className="sep">Organization</div>{ORG.map(link)}
      <div className="sep">System</div>{SYS.map(link)}
      <div className="foot">{sys ? <>v{sys.version}<br />{sys.liveSessions} live · dispatch {sys.autoDispatch ? 'auto' : 'manual'}<br /><a href="/m/" style={{ color: 'inherit' }}>Mobile UI ↗</a></> : '…'}</div>
    </aside>
    <div className="main">
      <div className="topbar"><h1>{crumb && <span className="crumb">{crumb} / </span>}{title}</h1>{right}
        <button type="button" className="search" onClick={() => setPal(true)}>Search, run a command…  ⌘K</button><Identity /></div>
      {tabs}
      {noPad ? children : <div className="content">{children}</div>}
    </div>
    {pal && <Palette onClose={() => setPal(false)} />}
  </div>;
}

export function ProjectTabs({ id, on }: { id: string; on: string }) {
  const t: [string, string][] = [['overview', 'Overview'], ['artifacts', 'Artifacts'], ['queue', 'Queue'], ['sessions', 'Sessions'], ['releases', 'Releases'], ['history', 'History'], ['settings', 'Settings']];
  return <nav className="tabs">{t.map(([k, l]) => <A key={k} to={k === 'overview' ? `/projects/${id}` : `/projects/${id}/${k}`} className={on === k ? 'on' : ''}>{l}</A>)}</nav>;
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
  return <div className="overlay" onClick={(e) => { if (e.target === e.currentTarget) onClose(); }}>
    <div className="palette" role="dialog" aria-label="Command palette">
      <input autoFocus placeholder="Search projects, tasks, docs, commands…" value={q} onChange={(e) => { setQ(e.target.value); setSel(0); }}
        onKeyDown={(e) => { if (e.key === 'Escape') onClose(); if (e.key === 'ArrowDown') setSel((s) => Math.min(hits.length - 1, s + 1)); if (e.key === 'ArrowUp') setSel((s) => Math.max(0, s - 1)); if (e.key === 'Enter' && hits[sel]) go(hits[sel]); }} />
      <div className="res">
        {hits.map((h, k) => <div key={k} className={'hit ' + (k === sel ? 'on' : '')} onClick={() => go(h)}>
          <span className="chip">{h.kind}</span><span className="grow">{h.title}</span><code className="muted">{h.cli}</code></div>)}
        {q && hits.length === 0 && <div className="empty">No matches</div>}
        {!q && <div className="empty small">Type to search. Commands show the matching <code>gbi</code> CLI call; Enter copies it.</div>}
      </div>
    </div>
  </div>;
}
