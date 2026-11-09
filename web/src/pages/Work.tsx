import { useEffect, useRef, useState } from 'react';
import { api } from '../api.gen';
import { useData, useLive } from '../live';
import { navigate, query } from '../router';
import { ProjectTabs, Shell, useSystem } from '../shell';
import { A, Av, Bar, Card, Chip, DictArea, Empty, Err, Loading, Seg, Status, Viewer, ago, clock, kfmt, useAction } from '../ui';
import { attachmentsFor } from './Artifacts';
import { clickable, useDialog } from '../keys';
import { LiveTerminal } from '../terminal';

// v04 · Questions dashboard (FR-100–FR-105, FR-102a, FR-107)
export function Questions({ qid }: { qid?: string }) {
  const { data, error } = useData<any[]>(() => api.questions.list({ status: 'open' }), [], (e) => e.type.startsWith('question'));
  const open = data || [];
  const { data: inbox } = useData<any[]>(() => api.inbox.list(), [], (e) => e.type.startsWith('session'));
  const waiting = (inbox || []).filter((n) => n.kind === 'member');
  const sel = qid || open[0]?.id || '';
  return <Shell title="Questions" right={<><Chip c="red">{open.filter((q) => q.blocking).length} blocking</Chip><Chip>{open.filter((q) => !q.blocking).length} non-blocking</Chip></>} noPad>
    <Err msg={error} />
    <div style={{ display: 'grid', gridTemplateColumns: 'minmax(260px, 380px) 1fr', minHeight: 'calc(100vh - 52px)' }}>
      <div style={{ borderRight: '1px solid var(--line)', background: 'var(--card)' }}>
        {waiting.length > 0 && <><div className="section-label">Members waiting for you — in their terminal</div>
          {waiting.map((n) => <div key={n.id} className="list-item" {...clickable(() => navigate('/sessions/' + n.id + '?view=terminal'), { role: 'link' })}>
            <div className="small muted">{n.projectId} · herdr · {ago(n.createdAt)}</div><b>{n.title}</b> <Chip c="red">blocked</Chip><div className="small muted">{n.detail}</div></div>)}</>}
        <div className="section-label">Blocking — work is paused</div>
        {open.filter((q) => q.blocking).map((q) => <QRow key={q.id} q={q} sel={q.id === sel} />)}
        <div className="section-label">Non-blocking — proceeding on an assumption</div>
        {open.filter((q) => !q.blocking).map((q) => <QRow key={q.id} q={q} sel={q.id === sel} />)}
        {open.length === 0 && <Empty>No open questions. 🎉</Empty>}
        <div style={{ padding: 10 }}><A to="/questions/answered" className="small">Answered questions ›</A></div>
      </div>
      <div className="content">{sel === 'answered' ? <Answered /> : sel ? <QuestionDetail key={sel} id={sel} /> : <Empty>Select a question.</Empty>}</div>
    </div>
  </Shell>;
}

function QRow({ q, sel }: { q: any; sel: boolean }) {
  return <div className={'list-item ' + (sel ? 'sel' : '')} {...clickable(() => navigate('/questions/' + q.id), { role: 'link' })}>
    <div className="small muted">{q.projectId} · {q.asker} · {ago(q.createdAt)}</div><b>{q.title}</b>
    <div className="small muted">{q.blocking ? (q.taskId ? `Blocks ${q.taskId}` : 'Blocking') : 'Assumed: ' + q.assumption}{q.attachments?.length ? ` · ${q.attachments.length} attachments` : ''}</div></div>;
}

function Answered() {
  const { data } = useData<any[]>(() => api.questions.list({ status: 'answered' }), []);
  return <Card title="Answered">{(data || []).map((q) => <div key={q.id} className="small" style={{ padding: '6px 0', borderBottom: '1px solid var(--line)' }}><b>{q.id} {q.title}</b><div className="muted">{q.answeredBy}: {q.choice && `chose ${q.choice}. `}{q.answer}</div></div>)}{data?.length === 0 && <Empty>None yet.</Empty>}</Card>;
}

function QuestionDetail({ id }: { id: string }) {
  const { data: q, error } = useData<any>(() => api.questions.get(id), [id], (e) => e.subject === id);
  const [choice, setChoice] = useState('');
  const [answer, setAnswer] = useState('');
  const [propose, setPropose] = useState(false);
  const [viewer, setViewer] = useState<number | null>(null);
  const { run, busy } = useAction();
  useEffect(() => { if (q && !choice) setChoice((q.options.find((o: any) => o.recommended) || {}).key || ''); }, [q, choice]);
  if (error) return <Err msg={error} />;
  if (!q) return <Loading />;
  const atts = attachmentsFor(q.projectId, q.attachments);
  const next = () => api.questions.list({ status: 'open' }).then((l: any[]) => navigate('/questions/' + (l[0]?.id || '')));
  if (q.status !== 'open') return <Card title={q.id}><b>{q.title}</b><p>{q.answeredBy} {q.status}: {q.choice && `chose ${q.choice}. `}{q.answer}</p></Card>;
  return <>
    <div className="row c"><Av name={q.asker} role="developer" /><div><b>{q.asker}</b> · {q.projectId} {q.blocking ? <Chip c="red">Blocking</Chip> : <Chip>Non-blocking</Chip>}
      <div className="small muted">Asked {ago(q.createdAt)}{q.sessionId && ` from session ${q.sessionId}`}{q.taskId && ` · Task ${q.taskId}`}{q.reqs?.length ? ` · Relates to ${q.reqs.join(', ')}` : ''}</div></div></div>
    <div style={{ fontSize: 18, fontWeight: 700 }}>{q.title}</div>
    {q.why && <Card title="Why it matters"><div style={{ whiteSpace: 'pre-wrap' }}>{q.why}</div></Card>}
    {!q.blocking && <div className="note">Work continues on this assumption: <b>{q.assumption}</b> <button type="button" className="btn sm" onClick={() => run(() => api.questions.confirm(q.id), 'Assumption confirmed').then(next)}>Confirm assumption</button></div>}
    <div className="row">
      {q.options.length > 0 && <Card title="Options" className="grow">
        {q.options.map((o: any) => <label key={o.key} className={'opt-label ' + (choice === o.key ? 'sel' : '')}>
          <input type="radio" name="opt" checked={choice === o.key} onChange={() => setChoice(o.key)} style={{ marginRight: 6 }} />
          <b>{o.key}. {o.title}</b> {o.recommended && <Chip c="acc">Recommended</Chip>}{o.detail && <div className="small muted">{o.detail}</div>}</label>)}
        {q.recommendation && <div className="small muted">Recommendation: {q.recommendation}</div>}
      </Card>}
      <Card title="Attachments" style={{ width: 300 }}>
        <div className="col small" style={{ gap: 8 }}>{atts.map((a, k) => <button type="button" className="lnk" key={k} onClick={() => setViewer(k)}>{a.kind === 'html' ? '🖼' : a.kind === 'diff' ? '±' : '📄'} {a.name} ›</button>)}
          {atts.length === 0 && <span className="muted">No attachments.</span>}{atts.length > 0 && <span className="muted">Click to open in the viewer</span>}
          {q.taskId && <A to={'/queue?task=' + q.taskId}>🔗 Task {q.taskId}</A>}{q.sessionId && <A to={'/sessions/' + q.sessionId}>🔗 Session {q.sessionId}</A>}</div>
      </Card>
    </div>
    <Card title={<>Your answer <span className="small muted" style={{ textTransform: 'none' }}>type, or 🎙 dictate and edit the transcript</span></>}>
      <DictArea value={answer} onChange={setAnswer} placeholder="Add context for the asker (optional when you pick an option)" />
      <div className="row c" style={{ marginTop: 8 }}><label className="small"><input type="checkbox" checked={propose} onChange={(e) => setPropose(e.target.checked)} /> Propose as project value</label>
        <span className="right" /><button type="button" className="btn" onClick={() => run(() => api.questions.skip(q.id, { why: 'skipped from dashboard' }), 'Skipped').then(next)}>Skip</button>
        <button type="button" className="btn p" disabled={busy || (!choice && !answer.trim())} onClick={() => run(() => api.questions.answer(q.id, { answer, choice, proposeValue: propose ? (answer || q.options.find((o: any) => o.key === choice)?.title || '') : '' }), q.blocking ? 'Answered: work resumes' : 'Answered').then((r) => r && next())}>
          {q.blocking ? 'Answer & resume work' : 'Answer'}</button></div>
    </Card>
    {viewer !== null && <Viewer items={atts} index={viewer} onClose={() => setViewer(null)} title={q.id} projectId={q.projectId} />}
  </>;
}

// v05 · Sessions, context monitor, handoffs (experimental) (FR-60, FR-65, FR-70–FR-74, NFR-11)
export function Sessions({ projectId }: { projectId?: string }) {
  const sys = useSystem();
  const [all, setAll] = useState(false);
  const { data, error } = useData<any[]>(() => api.sessions.list({ project: projectId, live: !all }), [projectId, all], (e) => e.type.startsWith('session'));
  const ss = data || [];
  const hx = sys?.experimental?.smartHandoffs;
  const body = <>
    <div className="row c"><Seg value={all ? 'all' : 'live'} onChange={(v) => setAll(v === 'all')} options={[['live', 'Live'], ['all', 'Recent']]} />
      <span className="small muted">Each member runs in herdr and keeps its conversation until context reaches {Math.round((sys?.experimental?.handoffThreshold || .75) * 100)}%, then rotates{hx ? ` · smart handoffs (experimental) on for ${(sys?.experimental?.smartHandoffProjects || []).join(', ') || 'all projects'}` : ''}. In a terminal: <code>herdr --session gbi</code></span></div>
    <Err msg={error} />
    <Card style={{ padding: '6px 14px' }}>{ss.length === 0 ? <Empty>{all ? 'No sessions yet.' : 'Nothing running.'} Run a task from the <A to="/queue">queue</A>.</Empty> :
      <table className="t"><tbody><tr><th>Member</th><th>Project · Task</th><th>Harness</th><th style={{ width: 220 }}>Context</th><th>When full</th><th>herdr</th><th /></tr>
        {[...ss].sort((a, b) => Number(b.status === 'blocked') - Number(a.status === 'blocked')).map((s) => <tr key={s.id} className="click" style={s.status === 'blocked' ? { background: 'var(--red-s)' } : undefined} {...clickable(() => navigate('/sessions/' + s.id + (s.status === 'blocked' ? '?view=terminal' : '')), { role: 'link' })}>
          <td><Av s name={s.memberId} role="developer" /> {s.memberId.split('@')[0]}</td><td>{s.projectId} · {s.taskId}</td><td>{s.harness}{s.model ? ` · ${s.model}` : ''}</td>
          <td><Bar pct={s.fraction * 100} warn={70} /><span className="small">{kfmt(s.used)} / {kfmt(s.window)} · {Math.round(s.fraction * 100)}%{s.measured ? '' : ' (est.)'}</span></td>
          <td className="small">{hx && (sys?.experimental?.smartHandoffProjects || []).concat([]).some((p: string) => p === s.projectId) ? <Chip c="blue">Handoff at {Math.round((sys?.experimental?.handoffThreshold || .75) * 100)}% · exp.</Chip> : `Auto-compact${s.compactions ? ` · ${s.compactions}× so far` : ''}`}</td>
          <td>{s.status === 'blocked' ? <Chip c="red">blocked · needs you</Chip> : s.live ? <Chip c="acc">{s.herdrState || 'working'}</Chip> : <Status s={s.status} />}{s.resumeAt && s.status === 'rate_limited' && <div className="small muted">resumes {clock(s.resumeAt)}</div>}</td>
          <td>{s.status === 'blocked' ? <A to={'/sessions/' + s.id + '?view=terminal'} className="btn p sm">Open terminal</A> : <A to={'/sessions/' + s.id}>Open</A>}</td></tr>)}</tbody></table>}
    </Card></>;
  if (projectId) return <Shell title="Sessions" crumb={<A to={`/projects/${projectId}`}>{projectId}</A>} tabs={<ProjectTabs id={projectId} on="sessions" />}>{body}</Shell>;
  return <Shell title="Sessions" right={<><Chip>{ss.filter((s) => s.live).length} running</Chip><Chip>{ss.filter((s) => s.status === 'rate_limited').length} rate-limited</Chip></>}>{body}</Shell>;
}

// v05/v05b · Session detail: structured and terminal views (FR-66), input (FR-61)
export function SessionDetail({ sid }: { sid: string }) {
  const { data: s } = useData<any>(() => api.sessions.get(sid), [sid], (e) => e.subject === sid);
  const [view, setView] = useState<'structured' | 'terminal'>((query('view') as any) || 'structured');
  const [lines, setLines] = useState<any[]>([]);
  const [input, setInput] = useState('');
  const box = useRef<HTMLDivElement>(null);
  const { run } = useAction();
  useEffect(() => { setLines([]); api.sessions.output(sid, { view }).then(setLines); }, [sid, view]);
  useLive((e) => {
    if (e.type === 'session.output' && e.subject === sid) {
      const kind = e.data?.kind;
      if (view === 'structured') setLines((l) => [...l, { seq: e.data.seq, ts: e.ts, kind, text: e.summary }]);
    }
  });
  useEffect(() => { box.current?.scrollTo(0, box.current.scrollHeight); }, [lines]);
  if (!s) return <Shell title={sid} crumb="Sessions"><Loading /></Shell>;
  return <Shell title={<>{s.id} · {s.memberId} · {s.taskId}</>} crumb={<A to="/sessions">Sessions</A>} right={<><Status s={s.status} />{s.measured ? <Chip c="grn">measured</Chip> : <Chip c="amb">estimated</Chip>}</>}>
    <div className="row c wrap">
      <Seg value={view} onChange={(v) => setView(v)} options={[['structured', 'Structured'], ['terminal', 'Terminal']]} />
      <span className="small muted">{s.harness}{s.model ? ` · ${s.model}` : ''} · {kfmt(s.used)} / {kfmt(s.window)} ({Math.round(s.fraction * 100)}%) · {s.compactions} compaction(s) · started {ago(s.startedAt)}</span>
      <span className="right" /><A to={`/queue?task=${s.taskId}`} className="btn">Task {s.taskId}</A>
      {s.live && <button type="button" className="btn d" onClick={() => { if (window.confirm('Cancel this session?')) run(() => api.sessions.cancel(sid), 'Cancelled'); }}>Cancel</button>}
    </div>
    <div style={{ width: '100%' }}><Bar pct={s.fraction * 100} warn={70} /></div>
    {view === 'terminal' && (s.paneId ? <LiveTerminal memberId={s.memberId} height={460} /> : <div className="note small">{s.memberId} has no herdr pane yet.</div>)}
    <div style={{ display: view === 'terminal' ? 'none' : undefined }}>
      <div className="termbar"><span>● {s.id} {s.harness}</span><span className="right">{view === 'terminal' ? 'herdr pane' : 'structured events from the transcript'}{s.live ? ' · live' : ''}</span></div>
      <div ref={box} className={view === 'terminal' ? 'term' : 'out'} style={{ height: 460, borderRadius: '0 0 6px 6px' }}>
        {lines.length === 0 && <span className="muted">No output yet.</span>}
        {view === 'terminal' ? null :
          lines.map((l) => <div key={l.seq} className={'k-' + l.kind}><span className="ts">{clock(l.ts)} </span>{l.kind === 'tool' ? '● ' : l.kind === 'input' ? '> ' : ''}{l.text}</div>)}
      </div>
    </div>
    {s.live ? <div className="row c"><div className="grow"><DictArea value={input} onChange={setInput} rows={2} placeholder="Send a message to the member (typed into its herdr pane)" /></div>
      <button type="button" className="btn p" disabled={!input} onClick={() => run(() => api.sessions.send(sid, { text: input }), 'Sent').then((r) => r && setInput(''))}>Send input</button></div>
      : <div className="small muted">Ended {ago(s.endedAt)}: {s.exitReason}{s.resumeAt && s.status === 'rate_limited' ? ` · resumes automatically at ${clock(s.resumeAt)}` : ''}</div>}
    {s.handoff && <Card title={<>Handoff briefing <Chip c="blue">Experimental</Chip></>}><pre style={{ whiteSpace: 'pre-wrap', margin: 0 }}>{s.handoff}</pre></Card>}
    <div className="note small">{s.memberId} runs in herdr{s.paneId ? <> (pane <span className="mono">{s.paneId}</span>)</> : null}. To watch or take control from a terminal: <code>herdr --session gbi</code>.{s.status === 'blocked' && <> <b>It is waiting for you</b>: answer in its terminal.</>}</div>
  </Shell>;
}

// v06 · Task queue (FR-75, FR-80–FR-82)
export function Queue({ projectId }: { projectId?: string }) {
  const project = projectId || query('project');
  const milestone = query('milestone');
  const focus = query('task');
  const [status, setStatus] = useState('');
  const { data, error } = useData<any[]>(() => api.tasks.list({ project, status, milestone }), [project, status, milestone], (e) => e.type.startsWith('task') || e.type.startsWith('session'));
  const { data: projects } = useData<any[]>(() => api.projects.list(), []);
  const { data: disp } = useData<any>(() => api.dispatcher.status(), [], (e) => e.type.startsWith('dispatcher') || e.type.startsWith('session'));
  const { data: split } = useData<any[]>(() => (project ? api.tasks.splitCandidates(project) : Promise.resolve([])), [project]);
  const [adding, setAdding] = useState(false);
  const [open, setOpen] = useState(focus);
  const { run } = useAction();
  const ts = data || [];
  const body = <>
    <div className="row c wrap">
      {!projectId && <select value={project} onChange={(e) => navigate('/queue' + (e.target.value ? '?project=' + e.target.value : ''))} aria-label="Project"><option value="">All projects</option>{(projects || []).map((p) => <option key={p.id} value={p.id}>{p.id}</option>)}</select>}
      <Seg value={status} onChange={setStatus} options={[['', 'All'], ['queued', 'Queued'], ['running', 'Running'], ['blocked', 'Blocked'], ['review', 'Review'], ['done', 'Done'], ['failed', 'Failed']]} />
      <span className="small muted">Concurrency: {disp?.live ?? 0} / {disp?.maxConcurrent ?? '?'} · dispatch {disp?.auto ? 'automatic' : 'manual'}</span>
      <span className="right" /><button type="button" className="btn" onClick={() => run(() => api.dispatcher.tick(), 'Dispatched ready tasks')}>▶ Dispatch ready</button>
      <button type="button" className="btn p" onClick={() => setAdding(true)}>+ Task</button>
    </div>
    {split && split.length > 0 && <div className="note">Pilot suggests splitting: {split.map((t: any) => <b key={t.id}>{t.id} ({t.compactions} compactions, {t.handoffs} handoffs) </b>)} — they repeatedly ran out of context (FR-75).</div>}
    <Err msg={error} />
    <Card style={{ padding: '6px 14px' }}>{ts.length === 0 ? <Empty>No tasks.</Empty> :
      <table className="t"><tbody><tr><th>ID</th><th>Task</th><th>Role · assignee</th><th>Pri</th><th>Depends on</th><th>Status</th><th>Retries · compactions</th><th /></tr>
        {ts.map((t) => <>
          <tr key={t.id} className={'click ' + (open === t.id ? 'sel' : '')} {...clickable(() => setOpen(open === t.id ? '' : t.id))} aria-expanded={open === t.id}>
            <td className="mono">{t.id}</td><td><b>{t.title}</b><div className="small muted">{t.projectId}{t.milestoneId ? ' · ' + t.milestoneId : ''}{t.blockedBy ? ' · ' + t.blockedBy : ''}</div></td>
            <td className="small">{t.role}{t.assignee ? ' · ' + t.assignee : ''}</td><td>P{t.priority}</td><td className="mono small">{t.deps.join(', ') || '—'}</td>
            <td><Status s={t.status} />{t.ready && t.status === 'queued' && <Chip c="acc">ready</Chip>}</td><td className="small">{t.retries}/{t.maxRetries} · {t.compactions}{t.compactions + t.handoffs >= 2 && <Chip c="amb">split?</Chip>}</td>
            <td onClick={(e) => e.stopPropagation()}>{(t.status === 'queued' || t.status === 'failed') && <button type="button" className="btn sm" onClick={() => run(() => api.tasks.run(t.id, {}), 'Session started')}>Run</button>}</td></tr>
          {open === t.id && <tr key={t.id + 'd'}><td colSpan={8}><TaskDetail id={t.id} /></td></tr>}
        </>)}</tbody></table>}
    </Card>
    {adding && <TaskDialog project={project} projects={projects || []} onClose={() => setAdding(false)} />}
  </>;
  if (projectId) return <Shell title="Queue" crumb={<A to={`/projects/${projectId}`}>{projectId}</A>} tabs={<ProjectTabs id={projectId} on="queue" />}>{body}</Shell>;
  return <Shell title="Task queue">{body}</Shell>;
}

function TaskDetail({ id }: { id: string }) {
  const { data } = useData<any>(() => api.tasks.get(id), [id], (e) => e.subject === id);
  const [note, setNote] = useState('');
  const { run } = useAction();
  if (!data) return <Loading />;
  const t = data.task;
  return <div className="row" style={{ padding: 6 }}>
    <div className="col grow" style={{ gap: 8 }}>
      {t.description && <div className="small" style={{ whiteSpace: 'pre-wrap' }}>{t.description}</div>}
      {t.contextNote && <div className="small"><b>Notes carried into the next session:</b><div style={{ whiteSpace: 'pre-wrap' }} className="muted">{t.contextNote}</div></div>}
      <div className="row c" style={{ gap: 6 }}><div className="grow"><DictArea value={note} onChange={setNote} rows={1} placeholder="Add a note for the next session…" /></div>
        <button type="button" className="btn" disabled={!note} onClick={() => run(() => api.tasks.update(id, { note }), 'Note added').then(() => setNote(''))}>Add note</button></div>
      <div className="row c wrap" style={{ gap: 6 }}>
        {t.branch && <span className="small mono">branch {t.branch}</span>}{t.prUrl && <a href={t.prUrl} target="_blank" rel="noreferrer" className="small">PR ↗</a>}
        {['queued', 'paused', 'done', 'failed'].filter((s) => s !== t.status).map((s) => <button type="button" key={s} className="btn sm" onClick={() => run(() => api.tasks.status(id, { status: s }), 'Status changed')}>Mark {s}</button>)}
        {t.status === 'review' && <A to="/reviews" className="btn sm p">Open review</A>}
      </div>
    </div>
    <div className="col small" style={{ width: 360, gap: 4 }}>
      <b>Sessions</b>{data.sessions.map((s: any) => <A key={s.id} to={'/sessions/' + s.id}>{s.id} · {s.memberId} · {s.status} · {Math.round(s.fraction * 100)}%</A>)}{data.sessions.length === 0 && <span className="muted">None yet</span>}
      <b style={{ marginTop: 6 }}>History</b>{data.history.slice(0, 6).map((e: any) => <span key={e.id} className="muted">{clock(e.ts)} {e.summary}</span>)}
    </div>
  </div>;
}

function TaskDialog({ project, projects, onClose }: { project: string; projects: any[]; onClose: () => void }) {
  const [t, setT] = useState<any>({ project: project || projects[0]?.id || '', title: '', description: '', role: 'developer', priority: 2, deps: '', reqs: '', milestone: '' });
  const { data: roles } = useData<any[]>(() => api.roles.list(), []);
  const { data: ms } = useData<any[]>(() => (t.project ? api.milestones.list(t.project) : Promise.resolve([])), [t.project]);
  const { data: members } = useData<any[]>(() => (t.project ? api.members.list(t.project) : Promise.resolve([])), [t.project]);
  const { run, busy } = useAction();
  const split = (s: string) => s.split(',').map((x) => x.trim()).filter(Boolean);
  const dlg = useRef<HTMLDivElement>(null);
  useDialog(dlg, onClose);
  return <div className="overlay" onClick={(e) => { if (e.target === e.currentTarget) onClose(); }}>
    <div className="modal" role="dialog" aria-modal="true" aria-label="New task" ref={dlg}>
      <h2 style={{ fontSize: 16 }}>Queue a task</h2>
      <div className="row"><div className="form-row grow"><label>Project</label><select value={t.project} onChange={(e) => setT({ ...t, project: e.target.value })}>{projects.map((p) => <option key={p.id}>{p.id}</option>)}</select></div>
        <div className="form-row"><label>Role</label><select value={t.role} onChange={(e) => setT({ ...t, role: e.target.value })}>{(roles || []).map((r) => <option key={r.id} value={r.id}>{r.name}</option>)}</select></div>
        <div className="form-row"><label>Assign to</label><select value={t.assignee || ''} onChange={(e) => setT({ ...t, assignee: e.target.value })}><option value="">any free {t.role} member</option>{(members || []).filter((m) => m.kind === 'ai' && m.role === t.role).map((m) => <option key={m.id} value={m.id}>{m.name}</option>)}</select></div>
        <div className="form-row"><label>Priority</label><select value={t.priority} onChange={(e) => setT({ ...t, priority: +e.target.value })}>{[1, 2, 3].map((p) => <option key={p} value={p}>P{p}</option>)}</select></div></div>
      <div className="form-row"><label>Title</label><input autoFocus value={t.title} onChange={(e) => setT({ ...t, title: e.target.value })} /></div>
      <div className="form-row"><label>Description and definition of done</label><DictArea value={t.description} onChange={(v) => setT({ ...t, description: v })} rows={4} /></div>
      <div className="row"><div className="form-row grow"><label>Requirements (comma separated)</label><input value={t.reqs} onChange={(e) => setT({ ...t, reqs: e.target.value })} placeholder="FR-21, FR-22" /></div>
        <div className="form-row grow"><label>Depends on</label><input value={t.deps} onChange={(e) => setT({ ...t, deps: e.target.value })} placeholder="T-12" /></div></div>
      <div className="form-row"><label>Milestone</label><select value={t.milestone} onChange={(e) => setT({ ...t, milestone: e.target.value })}><option value="">(none)</option>{(ms || []).map((m) => <option key={m.id} value={m.id}>{m.name}</option>)}</select></div>
      <div className="row c"><span className="right" /><button type="button" className="btn" onClick={onClose}>Cancel</button>
        <button type="button" className="btn p" disabled={busy || !t.title || !t.project} onClick={() => run(() => api.tasks.add({ ...t, deps: split(t.deps), reqs: split(t.reqs) }), 'Task queued').then((r) => r && onClose())}>Queue task</button></div>
    </div>
  </div>;
}
