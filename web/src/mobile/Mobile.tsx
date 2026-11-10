// Mobile web UI (FR-35): Approach A mobile, screens M1–M14.
import { useEffect, useRef, useState, type ReactNode } from 'react';
import { ChangelogList, UpdatePanel } from '../pages/Changelog';
import { api } from '../api.gen';
import { useData, useLive } from '../live';
import { match, navigate, usePath } from '../router';
import { useSystem } from '../shell';
import { Bar, Chip, DictArea, Empty, Loading, Markdown, Status, Viewer, ago, clock, day, kfmt, kindOf, useAction } from '../ui';
import { attachmentsFor } from '../pages/Artifacts';
import './mobile.css';
import { clickable, useGlobalKeys } from '../keys';
import { LiveTerminal } from '../terminal';
import { LessonSteps, useLessons, useLesson, type Lesson } from '../learn';

function Tabbar({ on }: { on: string }) {
  const { data: inbox } = useData<any[]>(() => api.inbox.list(), [], (e) => /^(question|review|rehearsal|options|value)\./.test(e.type));
  const n = inbox?.length || 0;
  const t: [string, string, string][] = [['projects', '▦', 'Projects'], ['inbox', '✉', 'Inbox'], ['sessions', '◉', 'Sessions'], ['more', '⋯', 'More']];
  return <nav className="m-tabbar">{t.map(([k, i, l]) => <button type="button" key={k} className={'lnk ' + (on === k ? 'on' : '')} onClick={() => navigate(k === 'projects' ? '/m/' : '/m/' + k)}><b>{i}</b>{l}{k === 'inbox' && n > 0 && <span className="m-dot">{n}</span>}</button>)}</nav>;
}

function Screen({ title, sub, back, tab, children, dark, foot }: { title: ReactNode; sub?: ReactNode; back?: string; tab?: string; children: ReactNode; dark?: boolean; foot?: ReactNode }) {
  const sys = useSystem();
  return <div className={'m-screen ' + (dark ? 'dark' : '')}>
    <header className="m-hdr">{back ? <button type="button" className="lnk m-sub" onClick={() => (back === 'back' ? history.back() : navigate(back))}>‹ Back</button> : <div className="m-sub">● {sys?.actor?.origin === 'tailscale' ? `Tailnet · ${sys.actor.login}` : `Local · ${sys?.actor?.name || ''}`}</div>}
      <h1>{title}</h1>{sub && <div className="m-sub">{sub}</div>}</header>
    <main className="m-body">{children}</main>
    {foot}
    {tab && <Tabbar on={tab} />}
  </div>;
}

// M1 · Projects
function Home() {
  const { data } = useData<any[]>(() => api.projects.list(), []);
  return <Screen title="Projects" tab="projects">
    {!data ? <Loading /> : data.length === 0 ? <Empty>No projects yet. Import one on desktop.</Empty> : data.map((p) => {
      const un = p.live?.unpushed || {};
      const attn = p.needsYou > 0 || un.commits > 0;
      return <div key={p.id} className={'m-card ' + (attn ? 'attn' : '')} {...clickable(() => navigate('/m/projects/' + p.id), { role: 'link' })}>
        <div className="m-row"><b>{p.name}</b><Chip c="blue">{p.stage}</Chip></div>
        <div className="small muted">{p.purpose}</div>
        <div className="small" style={{ marginTop: 4 }}>{p.blocking > 0 && <Chip c="red">{p.blocking} blocking</Chip>} {p.needsYou - p.blocking > 0 && <Chip c="amb">{p.needsYou - p.blocking} waiting</Chip>} {un.commits > 0 && <Chip c="red">⚠ {un.commits} unpushed</Chip>}</div>
        <div className="small" style={{ marginTop: 4 }}>{p.live?.release ? <b>{p.live.release.tag}</b> : <span className="muted">No stable release</span>} · {p.websiteUrl ? <a href={p.websiteUrl} onClick={(e) => e.stopPropagation()} target="_blank" rel="noreferrer">Site ↗</a> : <span className="muted">Site not published</span>}</div>
      </div>;
    })}
  </Screen>;
}

// M2 · Project status
function ProjectStatus({ id }: { id: string }) {
  const { data } = useData<any>(() => api.projects.get(id), [id], (e) => e.projectId === id);
  if (!data) return <Screen title={id} back="/m/" tab="projects"><Loading /></Screen>;
  const p = data.project;
  return <Screen title={p.name} sub={`${p.stage}${p.milestone ? ' · ' + p.milestone : ''}`} back="/m/" tab="projects">
    {data.needsYou.length > 0 && <div className="m-card" style={{ borderColor: 'var(--red)' }}><h4>Needs you</h4>{data.needsYou.map((n: any) => <div key={n.kind + n.id} className="small" style={{ margin: '4px 0' }} {...clickable(() => navigate(mlink(n)), { role: 'link' })}><Chip c={n.blocking ? 'red' : 'amb'}>{n.blocking ? 'Blocking' : n.kind}</Chip> {n.title} ›</div>)}</div>}
    <div className="m-card"><h4>Roadmap</h4>{data.milestones?.map((m: any) => <div key={m.id} className="small" style={{ margin: '4px 0' }}>{m.status === 'done' ? '✅' : m.status === 'active' ? '▶' : '○'} {m.name} {m.total ? `${m.done}/${m.total}` : ''}{m.gateName && <span className="muted"> · ◆ {m.gateName}</span>}{m.status === 'active' && <Bar pct={m.total ? (100 * m.done) / m.total : 0} />}</div>)}{!data.milestones?.length && <span className="muted small">No roadmap yet.</span>}</div>
    {(data.project?.team || []).length > 0 && <div className="m-card"><h4>Team</h4>{data.project.team.map((m: any) => <div key={m.id} className="small" style={{ margin: '3px 0' }}>{m.name} · <span className="muted">{m.kind === 'human' ? 'human · ' + m.role : m.role + (m.tuned ? ' · tuned' : '')}</span>{m.state ? <> · <Chip c={m.state === 'blocked' ? 'red' : m.state === 'working' ? 'acc' : ''}>{m.state}</Chip></> : null}</div>)}<div className="small muted">From archetypes, tuned for this project</div></div>}
    <div className="m-card"><h4>Running</h4>{data.sessions.map((s: any) => <div key={s.id} className="small" {...clickable(() => navigate('/m/sessions/' + s.id), { role: 'link' })} style={{ margin: '4px 0' }}>{s.memberId} · {s.taskId} · {Math.round(s.fraction * 100)}%{s.compactions ? ` · compacted ${s.compactions}×` : ''}<Bar pct={s.fraction * 100} warn={70} /></div>)}{data.sessions.length === 0 && <span className="muted small">Nothing running.</span>}</div>
    <div className="m-card" {...clickable(() => navigate(`/m/projects/${id}/docs`), { role: 'link' })}><h4>Documents ›</h4><div className="small muted">Requirements · Mockups · Plan · Tracker · Compare · Support files</div></div>
    <div className="m-card"><h4>Release & links</h4><div className="small">{p.live?.release ? <><b>{p.live.release.tag}</b> · {day(p.live.release.date)}</> : 'No stable release yet'} {p.live?.repo?.ci && p.live.repo.ci !== 'none' && <>· CI <Status s={p.live.repo.ci === 'passing' ? 'done' : 'failed'} /></>}<br />
      {p.live?.repo?.releaseUrl && <a href={p.live.repo.releaseUrl}>Notes ↗ </a>}{p.websiteUrl && <a href={p.websiteUrl}>Website ↗ </a>}{p.live?.repo?.repoUrl && <a href={p.live.repo.repoUrl}>Repo ↗</a>}</div></div>
  </Screen>;
}

function mlink(n: any) {
  return ({ question: '/m/q/' + n.id, review: '/m/review/' + n.id, gate: '/m/rehearsal/' + n.id, options: '/m/options/' + n.id, value: '/m/projects/' + n.projectId, member: '/m/sessions/' + n.id } as any)[n.kind] || '/m/inbox';
}

// M3 · Inbox (everything waiting on you, FR-104)
function Inbox() {
  const { data } = useData<any[]>(() => api.inbox.list(), [], (e) => /^(question|review|rehearsal|options|value|session)\./.test(e.type));
  const items = data || [];
  const group = (label: string, xs: any[]) => xs.length > 0 && <><div className="m-label">{label}</div>{xs.map((n) => <div key={n.kind + n.id} className="m-li" {...clickable(() => navigate(mlink(n)), { role: 'link' })}>
    <div><div className="small muted">{[n.projectId, n.from, ago(n.createdAt)].filter(Boolean).join(' · ')}</div><b>{n.title}</b>{n.detail && <div className="small muted">{n.detail.slice(0, 80)}</div>}</div><Chip c={n.blocking ? 'red' : n.kind === 'review' || n.kind === 'gate' ? 'amb' : 'blue'}>{n.kind === 'question' ? 'Q' : n.kind}</Chip></div>)}</>;
  return <Screen title="Inbox" sub={`${items.filter((i) => i.blocking).length} blocking · ${items.length} total`} tab="inbox">
    <div className="m-flush">{group('Blocking · work paused', items.filter((i) => i.blocking))}{group('Needs decision', items.filter((i) => !i.blocking && i.kind !== 'question'))}{group('Non-blocking · assumed', items.filter((i) => !i.blocking && i.kind === 'question'))}
      {data && items.length === 0 && <Empty>Nothing waiting on you.</Empty>}</div>
  </Screen>;
}

// M4 · Answer a question (FR-101–FR-105, FR-107)
function Answer({ id }: { id: string }) {
  const { data: q } = useData<any>(() => api.questions.get(id), [id]);
  const [choice, setChoice] = useState('');
  const [text, setText] = useState('');
  const [viewer, setViewer] = useState<number | null>(null);
  const { run, busy } = useAction();
  useEffect(() => { if (q && !choice) setChoice((q.options.find((o: any) => o.recommended) || {}).key || ''); }, [q, choice]);
  if (!q) return <Screen title="Question" back="/m/inbox"><Loading /></Screen>;
  const atts = attachmentsFor(q.projectId, q.attachments);
  return <Screen title={q.title} sub={<>{q.projectId} · {q.blocking ? <span style={{ color: '#ff9b93' }}>Blocking</span> : 'Non-blocking'}</>} back="/m/inbox">
    <div className="small"><b>{q.asker}</b>{q.taskId && ` · ${q.taskId}`}{q.reqs?.length ? ` · ${q.reqs.join(', ')}` : ''}</div>
    {q.why && <div className="m-card small"><h4>Why it matters</h4>{q.why}</div>}
    {!q.blocking && <div className="m-card small">Assumed: <b>{q.assumption}</b></div>}
    {q.options.map((o: any) => <div key={o.key} className={'m-opt ' + (choice === o.key ? 'sel' : '')} {...clickable(() => setChoice(o.key))}><b>{choice === o.key ? '◉' : '○'} {o.key}. {o.title}</b> {o.recommended && <Chip c="acc">Recommended</Chip>}{o.detail && <div className="small muted">{o.detail}</div>}</div>)}
    {atts.length > 0 && <div className="small">📎 {atts.map((a, k) => <button type="button" className="lnk" key={k} onClick={() => setViewer(k)}>{a.name} › </button>)}</div>}
    <DictArea value={text} onChange={setText} placeholder="Add a note (optional) — tap 🎙 to dictate" />
    <button type="button" className="m-btn p" disabled={busy || (!choice && !text)} onClick={() => run(() => api.questions.answer(id, { answer: text, choice }), 'Answered').then((r) => r && navigate('/m/inbox'))}>{q.blocking ? 'Answer & resume work' : 'Answer'}</button>
    {viewer !== null && <Viewer items={atts} index={viewer} onClose={() => setViewer(null)} title={q.id} projectId={q.projectId} />}
  </Screen>;
}

// M5 · Review (assessment + verdict, FR-110–FR-112)
function ReviewScreen({ id }: { id: string }) {
  const { data: r } = useData<any>(() => api.reviews.get(id), [id]);
  const [text, setText] = useState('');
  const [viewer, setViewer] = useState(false);
  const { run, busy } = useAction();
  if (!r) return <Screen title="Review" back="/m/inbox"><Loading /></Screen>;
  const a = r.assessment || {}, t = a.tests || {};
  const met = (xs: any[]) => `${(xs || []).filter((x) => x.status === 'met').length}/${(xs || []).length}`;
  const pair = (a.mockups || []).find((m: any) => m.mockup && m.built) || (a.mockups || [])[0];
  const items = pair ? [pair.mockup && { name: 'Mockup', kind: kindOf(pair.mockup), url: `/raw/${r.projectId}/${pair.mockup}` }, pair.built && { name: 'Built', kind: 'image', url: `/raw/${r.projectId}/${pair.built}` }].filter(Boolean) : [];
  const verdict = (v: string) => run(() => api.reviews.verdict(id, { verdict: v, comment: text }), 'Review submitted').then((x) => x && navigate('/m/inbox'));
  return <Screen title={`${r.taskId} ${r.taskTitle}`} sub={<>{r.projectId}{a.prUrl && <> · <a href={a.prUrl}>PR ↗</a></>} · Lens: {a.recommendation === 'changes' ? 'recommends changes' : 'ready'}</>} back="/m/inbox">
    {items.length > 0 && <div className="m-row" style={{ gap: 6 }} {...clickable(() => setViewer(true))}>
      <div className="m-shot">{pair?.mockup ? 'Mockup ⤢' : 'no mockup'}</div><div className="m-shot" style={{ background: 'var(--card)' }}>{pair?.built ? <img src={`/raw/${r.projectId}/${pair.built}`} alt="built" style={{ maxWidth: '100%', maxHeight: 90 }} /> : 'no screenshot'}</div></div>}
    {a.summary && <div className="small">{a.summary}</div>}
    <div className="m-card m-flush">
      <div className="m-li"><span>🧪 Unit {t.ran ? `${t.unitPassed}/${t.unitPassed + t.unitFailed}` : '—'} · E2E {t.e2ePassed + t.e2eFailed ? `${t.e2ePassed}/${t.e2ePassed + t.e2eFailed}` : '—'}</span><Chip c={t.unitFailed || t.e2eFailed ? 'red' : 'grn'}>{t.unitFailed || t.e2eFailed ? 'fail' : 'pass'}</Chip></div>
      <div className="m-li"><span>📐 Standards {met(a.standards)}</span></div>
      <div className="m-li"><span>💬 Feedback {met(a.feedback)}</span></div>
      <div className="m-li"><span>📋 Requirements {met(a.requirements)}</span></div>
    </div>
    {r.status === 'pending' ? <><DictArea value={text} onChange={setText} placeholder="Comment (required to request changes)" />
      <div className="m-row" style={{ gap: 8 }}><button type="button" className="m-btn d" disabled={busy} onClick={() => verdict('reject')}>Reject</button><button type="button" className="m-btn" disabled={busy || !text} onClick={() => verdict('changes')}>Changes</button><button type="button" className="m-btn p" disabled={busy} onClick={() => verdict('approve')}>Approve</button></div></>
      : <div className="small muted">{r.reviewer} {r.status} {ago(r.decidedAt)}</div>}
    {viewer && <Viewer items={items as any} index={0} onClose={() => setViewer(false)} title={r.id} projectId={r.projectId} />}
  </Screen>;
}

// M6 · Rehearsal step (FR-116–FR-118, FR-117a)
function RehearsalStep({ id }: { id: string }) {
  const { data: g } = useData<any>(() => api.rehearsals.get(id), [id], (e) => e.subject === id);
  const [text, setText] = useState('');
  const [viewer, setViewer] = useState(false);
  const { run, busy } = useAction();
  if (!g) return <Screen title="Rehearsal" back="/m/inbox"><Loading /></Screen>;
  const st = g.steps[g.current] || {};
  const fs = g.findings.filter((f: any) => f.step === g.current);
  const screen = st.screen ? (st.screen.startsWith('http') ? st.screen : `/raw/${g.projectId}/${st.screen}`) : '';
  return <Screen title={`${g.current + 1} · ${st.title || ''}`} sub={<>{g.projectId} · {g.id} · Step {g.current + 1} / {g.steps.length}<Bar pct={(100 * (g.current + 1)) / g.steps.length} /></>} back="/m/inbox">
    <div className="small">{st.scenario}</div>
    <div className="m-shot" style={{ height: 150 }} {...clickable(() => screen && setViewer(true))}>{screen ? 'Screen preview · tap for full size ⤢' : 'No screen for this step'}</div>
    <div className="small muted">Findings · {fs.length} from {g.reviewers.join(', ') || 'nobody yet'}</div>
    {fs.map((f: any) => <div key={f.id} className="small"><b>{f.author}</b>: {f.text} {f.changeRequest ? <Chip c="acc">{f.changeRequest}</Chip> : <button type="button" className="lnk" onClick={() => run(() => api.findings.changeRequest(String(f.id), { now: true }), 'Running now')}>Do it now</button>}</div>)}
    <DictArea value={text} onChange={setText} placeholder="Feedback on this step (🎙)" rows={2} />
    <div className="m-row" style={{ gap: 8 }}>
      <button type="button" className="m-btn" disabled={g.current === 0} onClick={() => run(() => api.rehearsals.advance(id, { step: g.current - 1 }))}>◀</button>
      <button type="button" className="m-btn" disabled={busy || !text} onClick={() => run(() => api.rehearsals.finding(id, { step: g.current, text }), 'Noted').then((r) => r && setText(''))}>Add note</button>
      <button type="button" className="m-btn p" onClick={() => run(() => api.rehearsals.advance(id, { step: Math.min(g.steps.length - 1, g.current + 1), status: 'ok' }))}>Good ▶</button></div>
    {g.status === 'open' && g.steps.every((s: any) => s.status !== 'pending') && <button type="button" className="m-btn p" onClick={() => run(() => api.rehearsals.decide(id, { decision: 'approve' }), 'Gate approved')}>Approve gate</button>}
    {viewer && <Viewer items={[{ name: st.title, kind: st.screen?.startsWith('http') ? 'html' : kindOf(st.screen || ''), url: screen }]} index={0} onClose={() => setViewer(false)} />}
  </Screen>;
}

// M7 · Option set
function OptionScreen({ id }: { id: string }) {
  const { data: s } = useData<any>(() => api.options.get(id), [id]);
  const [k, setK] = useState(0);
  const [tray, setTray] = useState<{ option: string; aspect: string }[]>([]);
  const [reason, setReason] = useState('');
  const [viewer, setViewer] = useState(false);
  const { run, busy } = useAction();
  if (!s) return <Screen title="Options" back="/m/inbox"><Loading /></Screen>;
  const o = s.options[k];
  const has = (a: string) => tray.some((t) => t.option === o.key && t.aspect === a);
  const tog = (a: string) => setTray(has(a) ? tray.filter((t) => !(t.option === o.key && t.aspect === a)) : [...tray, { option: o.key, aspect: a }]);
  return <Screen title={s.title} sub={`${s.projectId} · ${s.id} · option ${k + 1} of ${s.options.length}`} back="/m/inbox">
    <div className="m-shot" style={{ height: 220 }} {...clickable(() => o.mockup && setViewer(true))}>{o.mockup ? <iframe title={o.title} src={`/raw/${s.projectId}/${o.mockup}`} sandbox="allow-scripts" style={{ width: '100%', height: '100%', border: 0, pointerEvents: 'none', background: '#fff' }} /> : 'No visual'}</div>
    {o.mockup && <button type="button" className="lnk small" onClick={() => setViewer(true)}>⤢ Full screen</button>}
    <b>{o.key} · {o.title}</b><div className="small">{o.summary}</div>
    <div className="small">{(o.pros || []).map((p: string) => <div key={p} style={{ color: 'var(--grn)' }}>+ {p}</div>)}{(o.cons || []).map((c: string) => <div key={c} style={{ color: 'var(--red)' }}>− {c}</div>)}</div>
    <div className="m-card small"><h4>Keep these aspects</h4><label><input type="checkbox" checked={has('')} onChange={() => tog('')} /> Whole option</label>{(o.aspects || []).map((a: string) => <div key={a}><label><input type="checkbox" checked={has(a)} onChange={() => tog(a)} /> {a}</label></div>)}</div>
    <div className="m-row" style={{ gap: 8 }}><button type="button" className="m-btn" disabled={k === 0} onClick={() => setK(k - 1)}>◀ Prev</button><button type="button" className="m-btn" disabled={k === s.options.length - 1} onClick={() => setK(k + 1)}>Next ▶</button></div>
    <div className="small muted">In the tray: {tray.map((t) => `${t.option}${t.aspect ? ':' + t.aspect : ''}`).join(', ') || 'nothing yet'}</div>
    {s.status === 'open' ? <><DictArea value={reason} onChange={setReason} placeholder="Why this combination? (🎙)" rows={2} />
      <button type="button" className="m-btn p" disabled={busy || !tray.length || !reason} onClick={() => run(() => api.options.choose(id, { picks: tray, reason }), 'Choice recorded').then((r) => r && navigate('/m/inbox'))}>Submit choice</button></> : <div className="small muted">Decided: {s.reason}</div>}
    {viewer && o.mockup && <Viewer items={[{ name: o.title, kind: kindOf(o.mockup), url: `/raw/${s.projectId}/${o.mockup}` }]} index={0} onClose={() => setViewer(false)} />}
  </Screen>;
}

// M8 · Sessions
function SessionsScreen() {
  const sys = useSystem();
  const { data } = useData<any[]>(() => api.sessions.list({ live: true }), [], (e) => e.type.startsWith('session'));
  return <Screen title="Sessions" sub={`${data?.length || 0} running · auto-compact${sys?.experimental?.smartHandoffs ? ' · handoffs exp.' : ''}`} tab="sessions">
    {(data || []).map((s) => <div key={s.id} className="m-card" {...clickable(() => navigate('/m/sessions/' + s.id), { role: 'link' })}>
      <div className="m-row"><b>{s.memberId} · {s.projectId}</b><Status s={s.status} /></div><div className="small muted">{s.taskId} · {s.harness}</div>
      <Bar pct={s.fraction * 100} warn={70} /><div className="small">{Math.round(s.fraction * 100)}%{s.measured ? '' : ' (est.)'}{s.compactions ? ` · compacted ${s.compactions}×` : ''}{s.resumeAt && s.status === 'rate_limited' ? ` · resumes ${clock(s.resumeAt)}` : ''}</div>
      <div className="m-row" style={{ gap: 6, marginTop: 6 }}><span className="m-btn sm">Structured</span><span className="m-btn sm">⌨ Terminal ›</span></div></div>)}
    {data?.length === 0 && <Empty>Nothing running.</Empty>}
  </Screen>;
}

// M10 · Terminal view with key bar (FR-66, FR-67)
function Terminal({ sid }: { sid: string }) {
  const { data: s } = useData<any>(() => api.sessions.get(sid), [sid], (e) => e.subject === sid);
  const [view, setView] = useState<'terminal' | 'structured'>('terminal');
  const [lines, setLines] = useState<any[]>([]);
  const [input, setInput] = useState('');
  const box = useRef<HTMLDivElement>(null);
  const { run } = useAction();
  useEffect(() => { api.sessions.output(sid, { view }).then(setLines); }, [sid, view]);
  useLive((e) => { if (e.type === 'session.output' && e.subject === sid) setLines((l) => [...l, { seq: e.data.seq, text: e.summary, kind: e.data.kind }]); });
  useEffect(() => { box.current?.scrollTo(0, box.current.scrollHeight); }, [lines]);
  const key = (k: string) => setInput((v) => v + ({ Tab: '\t', Esc: '', '⏎': '' } as any)[k]);
  return <Screen title={`${s?.memberId || ''} · ${sid}`} sub={<div className="m-row" style={{ gap: 6 }}><span className="seg">{(['structured', 'terminal'] as const).map((v) => <button key={v} type="button" className={view === v ? 'on' : ''} onClick={() => setView(v)}>{v === 'terminal' ? 'Terminal' : 'Structured'}</button>)}</span>{s && <Chip c="acc">{s.live ? 'watching' : s.status}</Chip>}</div>} back="/m/sessions" dark
    foot={<><div className="m-keys">{['Esc', 'Tab', 'Ctrl', '↑', '↓', '⏎'].map((k) => <span key={k} {...clickable(() => key(k))}>{k}</span>)}</div>
      <div className="m-termfoot">{s?.live ? <div className="m-row" style={{ gap: 6 }}><input value={input} onChange={(e) => setInput(e.target.value)} placeholder="Send input to the agent" style={{ flex: 1 }} /><button type="button" className="m-btn p sm" onClick={() => run(() => api.sessions.send(sid, { text: input }), 'Sent').then((r) => r && setInput(''))}>Send</button></div>
        : <div className="small" style={{ color: '#6f8190', textAlign: 'center' }}>{s?.exitReason}</div>}
        <div className="small" style={{ color: '#6f8190', textAlign: 'center', marginTop: 6 }}>{s?.status === 'blocked' ? 'Waiting for you: answer in its herdr terminal' : 'Runs in herdr: herdr --session gbi'}</div></div></>}>
    {view === 'terminal' && s?.memberId ? <LiveTerminal memberId={s.memberId} height="60vh" keyBar compact />
      : <div ref={box} className="m-term">{lines.map((l) => <div key={l.seq} className={'k-' + l.kind}>{l.text}</div>)}{lines.length === 0 && <span style={{ color: '#6f8190' }}>No output yet.</span>}</div>}
  </Screen>;
}

// M9 · More
function More() {
  const sys = useSystem();
  const { data: docs } = useData<any[]>(() => api.docs.list(), []);
  const [doc, setDoc] = useState('');
  const { data: page } = useData<any>(() => (doc ? api.docs.get(doc) : Promise.resolve(null)), [doc]);
  const setTheme = (t: string) => { document.documentElement.setAttribute('data-theme', t); try { localStorage.setItem('gbi-theme', t); } catch { /* ignore */ } };
  if (page) return <Screen title={page.title} back="back"><Markdown src={page.body.replace(/\]\(screenshots\//g, '](/docs-assets/screenshots/')} /></Screen>;
  const upd = sys?.update?.updating ? ' · updating…' : sys?.update?.newer ? ` · update ready: ${sys.update.latest}` : '';
  return <Screen title="More" sub={<button type="button" className="lnk m-sub" style={{ color: 'inherit', padding: 0 }} onClick={() => navigate('/m/changelog')} data-testid="version-link">v{sys?.version || ''}{upd} ›</button>} tab="more">
    <div className="m-flush">
      <div className="m-li" {...clickable(() => navigate('/m/learn'))} data-testid="m-learn"><span>🎓 Learn</span><span className="muted">›</span></div>
      {[['👥 Team & personas', '/team'], ['📐 Global standards', '/standards'], ['☰ Task queue', '/queue'], ['🕘 History & insights', '/history'], ['⚙ Settings', '/settings'], ['🖥 Desktop UI', '/']].map(([l, to]) => <a key={to} className="m-li" href={to}><span>{l}</span><span className="muted">›</span></a>)}
      <div className="m-label">Docs</div>{(docs || []).map((d) => <div key={d.slug} className="m-li" {...clickable(() => setDoc(d.slug))}><span>📖 {d.title}</span><span className="muted">›</span></div>)}
      <div className="m-label">Theme</div><div className="m-li"><span className="seg"><button type="button" onClick={() => setTheme('light')}>Light</button><button type="button" onClick={() => setTheme('dark')}>Dark</button></span></div>
      <div style={{ padding: 12 }} className="small muted">Access: {sys?.actor?.origin} ({sys?.actor?.login || sys?.actor?.name}). Personas, standards and artifacts can be changed by chatting with an agent on desktop; the rich editor is desktop-first. Add this page to your home screen to install it.</div>
    </div>
  </Screen>;
}

// M12 · Change log & update (FR-06b–FR-06d), opened from the version on More.
function ChangelogScreen() {
  const sys = useSystem();
  const [check, setCheck] = useState<any>(undefined);
  return <Screen title="Change log" sub={sys ? `v${sys.version} installed` : ''} back="/m/more" tab="more">
    <div className="m-card"><UpdatePanel onCheck={setCheck} /></div>
    <div className="m-card"><ChangelogList check={check} /></div>
  </Screen>;
}

// M13 · Learn (FR-134–FR-136): lessons and progress; steps tick themselves off as on desktop.
function LearnScreen() {
  const { data, reload } = useLessons();
  const [open, setOpen] = useState('');
  const { data: one, setData } = useLesson(open || undefined);
  const { run } = useAction();
  const ls = data?.lessons || [];
  const done = ls.filter((l) => l.total > 0 && l.done === l.total).length;
  const start = async (l: Lesson) => { if (!l.started) await run(() => api.learn.start(l.slug)); setOpen(l.slug); reload(); };
  const l = one?.lesson;
  return <Screen title="Learn" sub={data ? `${done} of ${ls.length} done${data.practice ? ' · practice project: ' + data.practice.name : ''}` : ''} back="/m/more" tab="more">
    {!data ? <Loading what="the lessons" /> : <>
      {!data.practice && <div className="m-card small">Lessons use the practice project. <button type="button" className="btn p sm" onClick={() => run(() => api.learn.practice({}), 'Practice project created').then(reload)}>Create it</button></div>}
      {l && <div className="m-card small" data-testid="m-lesson"><h4 style={{ margin: '0 0 4px' }}>{l.done === l.total ? '✅' : '▶'} {l.number} · {l.title}</h4>
        <div className="bar"><i style={{ width: `${l.total ? (l.done / l.total) * 100 : 0}%` }} /></div>
        <LessonSteps lesson={l} practice={one!.practice} mobile onDone={(n) => run(() => api.learn.done(l.slug, { step: n })).then((d: any) => d && setData(d))} />
        {l.desktopOnly && <div className="muted" style={{ marginTop: 6 }}>Most of this lesson is on the desktop UI.</div>}</div>}
      <div className="m-flush">{ls.map((x) => <div key={x.slug} className="m-li" {...clickable(() => start(x))} data-lesson={x.slug}>
        <span>{x.total && x.done === x.total ? '✅ ' : x.started ? '▶ ' : ''}{x.number} · {x.title}</span><span className="muted small">{x.started && x.done < x.total ? `${Math.round((x.done / x.total) * 100)}%` : `${x.minutes} min`} ›</span></div>)}</div>
    </>}
  </Screen>;
}

// M14 · Project documents (FR-140, FR-141, FR-147)
const M_DOCS: [string, string, string][] = [['requirements', '📋', 'Requirements'], ['mockups', '🎨', 'Mockups'], ['plan', '🗺', 'Plan'], ['tracker', '✅', 'Tracker']];
function DocsScreen({ id }: { id: string }) {
  const { data: files } = useData<any>(() => api.artifacts.files(id), [id], (e) => e.type.startsWith('artifact'));
  const { data: cmp } = useData<any>(() => api.compare.list(id), [id], (e) => e.type.startsWith('artifact'));
  const STATE: Record<string, [string, string]> = { 'accepted': ['grn', 'accepted'], 'drifted': ['amb', 'drifted'], 'not-reviewed': ['red', 'not reviewed'], 'task': ['blue', 'task filed'], 'no-screenshot': ['', 'no screenshot'] };
  return <Screen title="Documents" sub={id} back={`/m/projects/${id}`} tab="projects">
    {!files ? <Loading /> : <>
      <div className="m-card">{M_DOCS.map(([r, i, l]) => <div key={r} className="m-row" style={{ padding: '7px 0' }} {...clickable(() => files.entries?.[r] && navigate(`/m/projects/${id}/docs/${r}`), { role: 'link' })}>
        <span>{i} <b>{l}</b></span><span className="small muted">{files.entries?.[r] ? files.entries[r].replace(/^docs\//, '') + ' ›' : 'not set'}</span></div>)}
        <div className="m-row" style={{ padding: '7px 0' }}><span>📁 <b>Support files</b></span><span className="small muted">{files.files?.length || 0}{files.unlabelled?.length ? ` · ${files.unlabelled.length} unlabelled` : ''}</span></div></div>
      <div className="m-card"><h4>Compare {cmp?.toReview > 0 && <Chip c="red">{cmp.toReview} to review</Chip>}</h4>
        {(cmp?.views || []).map((v: any) => <div key={v.id} className="m-row small" style={{ padding: '6px 0' }} {...clickable(() => navigate(`/m/projects/${id}/compare/${v.id}`), { role: 'link' })}><span>{v.title && v.title.startsWith(v.id) ? v.title : `${v.id} ${v.title || ''}`}</span><Chip c={STATE[v.state]?.[0]}>{STATE[v.state]?.[1]}</Chip></div>)}
        {!cmp?.views?.length && <span className="small muted">No mockup views yet.</span>}</div>
    </>}
  </Screen>;
}

function DocScreen({ id, role }: { id: string; role: string }) {
  const { data: files } = useData<any>(() => api.artifacts.files(id), [id]);
  const entry = files?.entries?.[role];
  return <Screen title={M_DOCS.find((d) => d[0] === role)?.[2] || role} sub={entry} back={`/m/projects/${id}/docs`}>
    {!files ? <Loading /> : !entry ? <Empty>No entry point yet. Set one on desktop or with gbi artifacts entry.</Empty>
      : <iframe className="m-docframe" src={`/raw/${id}/${entry}`} title={entry} sandbox="allow-scripts allow-forms allow-popups allow-popups-to-escape-sandbox" />}
  </Screen>;
}

function CompareScreen({ id, view }: { id: string; view: string }) {
  const { data: cmp } = useData<any>(() => api.compare.list(id), [id], (e) => e.type.startsWith('artifact'));
  const [side, setSide] = useState<'mock' | 'real'>('mock');
  const [note, setNote] = useState('');
  const { run, busy } = useAction();
  const v = (cmp?.views || []).find((x: any) => x.id === view);
  return <Screen title={v ? (v.title && v.title.startsWith(v.id) ? v.title : `${v.id} ${v.title || ''}`) : view} sub="Compare" back={`/m/projects/${id}/docs`}>
    {!cmp ? <Loading /> : !v ? <Empty>No view {view}.</Empty> : <>
      <div className="m-row" style={{ justifyContent: 'center', gap: 6, margin: '4px 0 8px' }}><button type="button" className={'btn sm ' + (side === 'mock' ? 'p' : '')} onClick={() => setSide('mock')}>Mockup</button><button type="button" className={'btn sm ' + (side === 'real' ? 'p' : '')} onClick={() => setSide('real')}>Real app</button></div>
      {side === 'mock' ? <iframe className="m-docframe" src={`/raw/${id}/${v.file}?gbi-view=${encodeURIComponent(v.id)}`} title={`Mockup ${v.id}`} sandbox="allow-scripts" />
        : v.screenshot ? <img src={`/raw/${id}/${v.screenshot}`} alt={`Screenshot of ${v.id}`} style={{ width: '100%', borderRadius: 10, border: '1px solid var(--line)' }} /> : <Empty>No screenshot yet.</Empty>}
      {v.screenshot && <div className="m-card" style={{ marginTop: 8 }}><DictArea value={note} onChange={setNote} rows={2} placeholder="What is different, or why it is fine" />
        <div className="m-row" style={{ gap: 6, marginTop: 6 }}><button type="button" className="btn" disabled={busy} onClick={() => run(() => api.compare.accept(id, v.id, { note }), 'Accepted')}>✓ Accept</button><button type="button" className="btn p" disabled={busy || !note.trim()} onClick={() => run(() => api.compare.task(id, v.id, { note }), 'Task filed')}>File a task</button></div></div>}
    </>}
  </Screen>;
}

export function MobileApp() {
  const path = usePath();
  useGlobalKeys();
  const routes: [string, (p: any) => JSX.Element][] = [
    ['/m', () => <Home />], ['/m/projects/:id', (p) => <ProjectStatus id={p.id} />], ['/m/projects/:id/docs', (p) => <DocsScreen id={p.id} />], ['/m/projects/:id/docs/:role', (p) => <DocScreen id={p.id} role={p.role} />], ['/m/projects/:id/compare/:view', (p) => <CompareScreen id={p.id} view={p.view} />], ['/m/inbox', () => <Inbox />], ['/m/q/:id', (p) => <Answer id={p.id} />],
    ['/m/review/:id', (p) => <ReviewScreen id={p.id} />], ['/m/rehearsal/:id', (p) => <RehearsalStep id={p.id} />], ['/m/options/:id', (p) => <OptionScreen id={p.id} />],
    ['/m/sessions', () => <SessionsScreen />], ['/m/sessions/:sid', (p) => <Terminal sid={p.sid} />], ['/m/more', () => <More />], ['/m/changelog', () => <ChangelogScreen />], ['/m/learn', () => <LearnScreen />],
  ];
  for (const [pat, r] of routes) { const m = match(pat, path.replace(/\/$/, '') || '/m'); if (m) return <div className="m-root">{r(m)}</div>; }
  return <div className="m-root"><Home /></div>;
}

export { kfmt };
