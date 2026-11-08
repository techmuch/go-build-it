import { useState } from 'react';
import { api } from '../api.gen';
import { useData } from '../live';
import { navigate } from '../router';
import { ProjectTabs, Shell } from '../shell';
import { A, Av, Bar, Card, Chip, DictArea, Empty, Err, Loading, Status, ago, day, useAction } from '../ui';

const STAGES = ['Discovery', 'Requirements', 'Design', 'Implementation', 'Review', 'Released', 'Maintenance', 'Paused', 'Archived'];
const stageColor = (s: string) => ({ Discovery: 'blue', Requirements: 'blue', Design: 'blue', Implementation: 'acc', Review: 'amb', Released: 'grn', Maintenance: '', Paused: '', Archived: '' } as any)[s] || '';

// v01 · Project catalog (FR-40–FR-44, FR-94)
export function Catalog() {
  const { data, error, loading, reload } = useData<any[]>(() => api.projects.list(), [], (e) => !e.type.startsWith('chat'));
  const [filter, setFilter] = useState<'all' | 'needs' | 'active' | 'paused'>('all');
  const [stage, setStage] = useState('');
  const [q, setQ] = useState('');
  const [sort, setSort] = useState<'attention' | 'name' | 'updated'>('attention');
  const [importing, setImporting] = useState(false);
  const { run, busy } = useAction();
  let ps = data || [];
  const needs = ps.filter((p) => p.needsYou > 0 || p.live?.unpushed?.commits > 0 || p.live?.unpushed?.uncommitted > 0 || p.live?.repo?.ci === 'failing');
  if (filter === 'needs') ps = needs;
  if (filter === 'active') ps = ps.filter((p) => !['Paused', 'Archived'].includes(p.stage));
  if (filter === 'paused') ps = ps.filter((p) => p.stage === 'Paused');
  if (stage) ps = ps.filter((p) => p.stage === stage);
  if (q) ps = ps.filter((p) => (p.id + ' ' + p.purpose).toLowerCase().includes(q.toLowerCase()));
  if (sort === 'name') ps = [...ps].sort((a, b) => a.name.localeCompare(b.name));
  if (sort === 'updated') ps = [...ps].sort((a, b) => (b.updatedAt || '').localeCompare(a.updatedAt || ''));
  return <Shell title="Projects" right={<button type="button" className="btn p" onClick={() => setImporting(true)}>+ Import project</button>}>
    <div className="row c wrap">
      {([['needs', `Needs you · ${needs.length}`, 'red'], ['all', `All · ${(data || []).length}`, ''], ['active', 'Active', ''], ['paused', 'Paused', '']] as const).map(([k, l, c]) =>
        <button type="button" key={k} className={'chip ' + (filter === k ? 'acc' : c)} style={{ border: 0, cursor: 'pointer', fontSize: 12, padding: '3px 10px' }} onClick={() => setFilter(k)}>{l}</button>)}
      <select value={stage} onChange={(e) => setStage(e.target.value)} aria-label="Stage"><option value="">Stage: all</option>{STAGES.map((s) => <option key={s}>{s}</option>)}</select>
      <select value={sort} onChange={(e) => setSort(e.target.value as any)} aria-label="Sort"><option value="attention">Sort: needs you first</option><option value="name">Sort: name</option><option value="updated">Sort: recently updated</option></select>
      <input placeholder="Filter projects…" value={q} onChange={(e) => setQ(e.target.value)} style={{ width: 200 }} />
      <button type="button" className="btn right" disabled={busy} onClick={() => run(() => api.projects.list({ refresh: true }).then(reload), 'Refreshed git and forge data')}>↻ Refresh</button>
    </div>
    <Err msg={error} />
    {loading && !data ? <Loading what="projects" /> : ps.length === 0 ? <Empty>{(data || []).length === 0 ? <>No projects yet. <a onClick={() => setImporting(true)}>Import a git repository</a> to start.</> : 'No projects match.'}</Empty> :
      <div className="pgrid">{ps.map((p) => <ProjectCard key={p.id} p={p} />)}</div>}
    {importing && <ImportDialog onClose={() => setImporting(false)} onDone={(id) => { setImporting(false); navigate('/projects/' + id); }} />}
  </Shell>;
}

function ProjectCard({ p }: { p: any }) {
  const l = p.live || {};
  const rel = l.release;
  const un = l.unpushed || {};
  const attn = p.needsYou > 0 || un.commits > 0 || un.uncommitted > 0 || l.repo?.ci === 'failing';
  const web = p.websiteUrl || l.repo?.websiteUrl;
  return <div className={'pcard ' + (attn ? 'attn' : '')} onClick={() => navigate('/projects/' + p.id)} role="link" tabIndex={0} onKeyDown={(e) => { if (e.key === 'Enter') navigate('/projects/' + p.id); }}>
    <div className="row c"><span className="nm">{p.name}</span><span className="right"><Chip c={stageColor(p.stage)}>{p.stage}</Chip></span></div>
    <div className="small">{p.purpose || <span className="muted">No purpose written yet</span>}</div>
    <div className="small muted">{p.statusLine}{p.milestone ? ` · ${p.milestone}` : ''}</div>
    <div className="row c wrap" style={{ gap: 6 }}>
      {p.blocking > 0 && <Chip c="red">{p.blocking} blocking</Chip>}
      {p.needsYou - p.blocking > 0 && <Chip c="amb">{p.needsYou - p.blocking} waiting on you</Chip>}
      {un.commits > 0 && <Chip c="red" title={(un.branches || []).join(', ')}>⚠ {un.commits} unpushed commit{un.commits > 1 ? 's' : ''}</Chip>}
      {un.uncommitted > 0 && <Chip c="amb">⚠ {un.uncommitted} uncommitted</Chip>}
      {l.repo?.ci === 'failing' && <Chip c="red">CI failing</Chip>}
      {(p.values || []).slice(0, 2).map((v: string) => <Chip key={v}>{v}</Chip>)}
    </div>
    <div className="stats">
      <div>Latest stable<b>{rel ? rel.tag : '—'}</b>{rel ? `${day(rel.date)} · ${rel.commitsSince} since` : 'No stable release yet'}</div>
      <div>CI<b>{l.repo?.ci && l.repo.ci !== 'none' ? l.repo.ci : '—'}</b>{l.repo?.ciAt ? ago(l.repo.ciAt) : ''}</div>
      <div>Open PRs / issues<b>{p.forge === 'github' ? `${l.repo?.openPrs ?? 0} / ${l.repo?.openIssues ?? 0}` : '—'}</b></div>
      <div>Sessions<b>{p.running} running</b>{p.queued} queued</div>
    </div>
    <div className="links" onClick={(e) => e.stopPropagation()}>
      {rel && l.repo?.releaseUrl ? <a href={l.repo.releaseUrl} target="_blank" rel="noreferrer">Release notes ↗</a> : <span className="muted">No stable release yet</span>}
      {web ? <a href={web} target="_blank" rel="noreferrer">Website ↗</a> : <span className="muted">Website not published</span>}
      {l.repo?.repoUrl && <a href={l.repo.repoUrl} target="_blank" rel="noreferrer">Repo ↗</a>}
      <A to={`/projects/${p.id}/artifacts`}>Docs</A>
    </div>
  </div>;
}

function ImportDialog({ onClose, onDone }: { onClose: () => void; onDone: (id: string) => void }) {
  const [path, setPath] = useState('');
  const [id, setId] = useState('');
  const [purpose, setPurpose] = useState('');
  const [stage, setStage] = useState('Discovery');
  const { run, busy } = useAction();
  return <div className="overlay" onClick={(e) => { if (e.target === e.currentTarget) onClose(); }}>
    <div className="modal" role="dialog" aria-label="Import project">
      <h2 style={{ fontSize: 16 }}>Import an existing git repository</h2>
      <div className="form-row"><label>Repository path on this machine</label><input autoFocus value={path} onChange={(e) => setPath(e.target.value)} placeholder="/Users/you/projects/trailmap" /></div>
      <div className="form-row"><label>Project id (optional; defaults to the folder name)</label><input value={id} onChange={(e) => setId(e.target.value)} /></div>
      <div className="form-row"><label>What is it? (one line)</label><DictArea value={purpose} onChange={setPurpose} rows={2} /></div>
      <div className="form-row"><label>Stage</label><select value={stage} onChange={(e) => setStage(e.target.value)}>{STAGES.map((s) => <option key={s}>{s}</option>)}</select></div>
      <div className="row c"><span className="muted small">CLI: <code>gbi projects import --path …</code></span><span className="right" /><button type="button" className="btn" onClick={onClose}>Cancel</button>
        <button type="button" className="btn p" disabled={busy || !path} onClick={() => run(() => api.projects.import({ path, id, purpose, stage })).then((p) => p && onDone(p.id))}>Import</button></div>
    </div>
  </div>;
}

// v02 · Project overview (FR-41, FR-44, FR-115, NFR-11)
export function ProjectOverview({ id }: { id: string }) {
  const { data, error } = useData<any>(() => api.projects.get(id), [id], (e) => e.projectId === id || e.type.startsWith('session'));
  const [editPurpose, setEditPurpose] = useState<string | null>(null);
  const [newM, setNewM] = useState(false);
  const { run } = useAction();
  if (error) return <Shell title={id} crumb="Projects"><Err msg={error} /></Shell>;
  if (!data) return <Shell title={id} crumb="Projects"><Loading /></Shell>;
  const p = data.project, l = p.live || {};
  return <Shell title={p.name} crumb={<A to="/">Projects</A>} right={<Chip c={stageColor(p.stage)}>{p.stage}</Chip>} tabs={<ProjectTabs id={id} on="overview" />}>
    <div className="row">
      <div className="col grow">
        <Card title="Purpose & values" right={<a onClick={() => setEditPurpose(editPurpose === null ? p.purpose : null)}>Edit</a>}>
          {editPurpose !== null ? <div className="col" style={{ gap: 6 }}><DictArea value={editPurpose} onChange={setEditPurpose} rows={2} />
            <div className="row c"><span className="right" /><button type="button" className="btn p" onClick={() => run(() => api.projects.update(id, { purpose: editPurpose }), 'Saved').then(() => setEditPurpose(null))}>Save</button></div></div>
            : <div style={{ fontSize: 14, marginBottom: 8 }}>{p.purpose || <span className="muted">No purpose yet.</span>}</div>}
          <div className="row c wrap" style={{ gap: 6 }}>{data.values.map((v: any) => <Chip key={v.id} c={v.status === 'proposed' ? 'acc' : ''} title={v.source}>{v.status === 'proposed' ? 'Proposed: ' : ''}{v.text}</Chip>)}
            <A to={`/projects/${id}/artifacts`} className="small">Values & problem ›</A></div>
        </Card>
        <Card title="Roadmap" right={<span className="small muted">◆ = review gate · click a row for detail · <a onClick={() => setNewM(true)}>+ Milestone</a></span>}>
          {data.milestones?.length ? <table className="t"><tbody>
            <tr><th style={{ width: '36%' }}>Milestone</th><th>Progress</th><th style={{ width: '24%' }}>Gate</th><th style={{ width: 90 }}>Status</th></tr>
            {data.milestones.map((m: any) => <tr key={m.id} className="click" onClick={() => navigate('/milestones/' + m.id)}>
              <td><A to={'/milestones/' + m.id}><b>{m.name}</b> ›</A></td>
              <td><Bar pct={m.total ? (100 * m.done) / m.total : m.status === 'done' ? 100 : 0} /><span className="small muted">{m.done} / {m.total} tasks</span></td>
              <td>{m.gateName ? <>◆ {m.gateName} <Status s={m.gateStatus} /></> : <span className="muted">—</span>}</td><td><Status s={m.status} /></td></tr>)}
          </tbody></table> : <Empty>No roadmap yet. <a onClick={() => setNewM(true)}>Add the first milestone</a>.</Empty>}
        </Card>
        <Card title="Running now">
          {data.sessions?.length ? <table className="t"><tbody>{data.sessions.map((s: any) => <tr key={s.id} className="click" onClick={() => navigate('/sessions/' + s.id)}>
            <td style={{ width: 30 }}><Av name={s.memberId} role="developer" /></td><td><b>{s.memberId}</b> · {s.harness}<br /><span className="small muted">{s.taskId}</span></td>
            <td style={{ width: 210 }}><Bar pct={s.fraction * 100} warn={70} /><span className="small muted">{Math.round(s.fraction * 100)}% context{s.measured ? '' : ' (est.)'}{s.compactions ? ` · compacted ${s.compactions}×` : ''}</span></td><td><Status s={s.status} /></td></tr>)}</tbody></table>
            : <span className="muted small">Nothing running. Queue tasks and run them, or turn on automatic dispatch in Settings.</span>}
        </Card>
      </div>
      <div className="col" style={{ width: 330 }}>
        <Card title="Needs you" style={{ borderColor: data.needsYou.length ? 'var(--red)' : undefined }}>
          {data.needsYou.length === 0 ? <span className="muted small">Nothing waiting on you.</span> :
            <div className="col small" style={{ gap: 8 }}>{data.needsYou.slice(0, 6).map((n: any) => <div key={n.kind + n.id}><Chip c={n.blocking ? 'red' : n.kind === 'review' ? 'amb' : ''}>{n.blocking ? 'Blocking' : n.kind}</Chip> {n.title} <A to={inboxLink(n)}>Open</A></div>)}</div>}
        </Card>
        <Card title="Releases">
          <dl className="kv"><dt>Latest stable</dt><dd>{l.release ? <><b>{l.release.tag}</b> · {day(l.release.date)}</> : 'No stable release yet'}</dd>
            <dt>Since release</dt><dd>{l.release ? `${l.release.commitsSince} commits` : '—'}</dd>
            <dt>Notes</dt><dd>{l.repo?.releaseUrl ? <a href={l.repo.releaseUrl} target="_blank" rel="noreferrer">Release notes ↗</a> : '—'}</dd>
            <dt>Website</dt><dd>{p.websiteUrl || l.repo?.websiteUrl ? <a href={p.websiteUrl || l.repo.websiteUrl} target="_blank" rel="noreferrer">{p.websiteUrl || l.repo.websiteUrl} ↗</a> : 'Website not published'}</dd></dl>
          <div style={{ marginTop: 8 }}><A to={`/projects/${id}/releases`} className="small">Release workflow ›</A></div>
        </Card>
        <Card title="Repository">
          <dl className="kv"><dt>Forge</dt><dd>{p.forge || 'none (local only)'}</dd><dt>Remote</dt><dd className="mono">{p.remote || '—'}</dd>
            <dt>Workflow</dt><dd>{wfText(data.workflow)}</dd><dt>CI</dt><dd>{l.repo?.ci && l.repo.ci !== 'none' ? <Chip c={l.repo.ci === 'passing' ? 'grn' : l.repo.ci === 'failing' ? 'red' : 'amb'}>{l.repo.ci}</Chip> : '—'}</dd>
            <dt>Unpushed</dt><dd>{l.unpushed?.commits || l.unpushed?.uncommitted ? <Chip c="red">{l.unpushed.commits} commits · {l.unpushed.uncommitted} files</Chip> : 'None'}</dd><dt>Path</dt><dd className="mono">{p.path}</dd></dl>
        </Card>
        {p.teamMembers?.length > 0 && <Card title="Team"><div className="row c wrap" style={{ gap: 4 }}>{p.teamMembers.map((m: string) => <Av key={m} name={m} />)}</div></Card>}
      </div>
    </div>
    {newM && <MilestoneDialog projectId={id} onClose={() => setNewM(false)} />}
  </Shell>;
}

export function wfText(w: any) {
  if (!w) return '—';
  return ['Branch per task', w.requirePr && 'PR/MR', w.lensReview && 'Lens review', w.humanApproval && 'human approves', w.squash ? 'squash-merge' : 'merge', w.tagReleases && 'CalVer tag', w.pages && 'Pages'].filter(Boolean).join(' → ');
}

export function inboxLink(n: any) {
  switch (n.kind) {
    case 'question': return '/questions/' + n.id;
    case 'review': return '/reviews/' + n.id;
    case 'gate': return '/rehearsals/' + n.id;
    case 'options': return '/options/' + n.id;
    case 'value': return `/projects/${n.projectId}/artifacts`;
  }
  return '/';
}

function MilestoneDialog({ projectId, existing, onClose }: { projectId: string; existing?: any; onClose: () => void }) {
  const [m, setM] = useState<any>(existing || { name: '', scope: '', gateName: '', gateKind: 'tabletop', gateCriteria: [] });
  const [crit, setCrit] = useState((existing?.gateCriteria || []).join('\n'));
  const { run, busy } = useAction();
  return <div className="overlay" onClick={(e) => { if (e.target === e.currentTarget) onClose(); }}>
    <div className="modal" role="dialog" aria-label="Milestone">
      <h2 style={{ fontSize: 16 }}>{existing ? 'Edit milestone' : 'New milestone'}</h2>
      <div className="form-row"><label>Name</label><input autoFocus value={m.name} onChange={(e) => setM({ ...m, name: e.target.value })} placeholder="M2 · Harness adapters" /></div>
      <div className="form-row"><label>Scope</label><DictArea value={m.scope} onChange={(v) => setM({ ...m, scope: v })} /></div>
      <div className="row"><div className="form-row grow"><label>Review gate (optional)</label><input value={m.gateName} onChange={(e) => setM({ ...m, gateName: e.target.value })} placeholder="Adapter demo" /></div>
        <div className="form-row"><label>Gate type</label><select value={m.gateKind} onChange={(e) => setM({ ...m, gateKind: e.target.value })}><option value="review">Review</option><option value="tabletop">Tabletop rehearsal</option><option value="uat">UAT rehearsal</option></select></div></div>
      <div className="form-row"><label>Gate passes when (one per line)</label><textarea value={crit} onChange={(e) => setCrit(e.target.value)} /></div>
      <div className="row c"><span className="right" /><button type="button" className="btn" onClick={onClose}>Cancel</button>
        <button type="button" className="btn p" disabled={busy || !m.name} onClick={() => run(() => api.milestones.save(projectId, { ...m, gateCriteria: crit.split('\n').map((s: string) => s.trim()).filter(Boolean) }), 'Milestone saved').then((r) => r && onClose())}>Save</button></div>
    </div>
  </div>;
}

// v02b · Roadmap item detail (FR-45b, FR-115)
export function MilestoneDetail({ mid }: { mid: string }) {
  const { data, error } = useData<any>(() => api.milestones.get(mid), [mid]);
  const [edit, setEdit] = useState(false);
  const { run } = useAction();
  if (error) return <Shell title={mid}><Err msg={error} /></Shell>;
  if (!data) return <Shell title={mid}><Loading /></Shell>;
  const m = data.milestone;
  const done = data.tasks.filter((t: any) => t.status === 'done').length;
  return <Shell title={m.name} crumb={<><A to={`/projects/${m.projectId}`}>{m.projectId}</A> / Roadmap</>} right={<Status s={m.status} />} tabs={<ProjectTabs id={m.projectId} on="overview" />}>
    <div className="row">
      <div className="col grow">
        <Card title="Scope" right={<a onClick={() => setEdit(true)}>Edit</a>}>
          <div>{m.scope || <span className="muted">No scope written.</span>}</div>
          <div className="row c" style={{ marginTop: 10 }}><div className="grow"><Bar pct={data.tasks.length ? (100 * done) / data.tasks.length : 0} /><span className="small muted">{done} of {data.tasks.length} tasks done</span></div>{m.target && <span className="small muted">target {m.target}</span>}</div>
          {!data.workAllowed && <div className="note" style={{ marginTop: 10 }}>Work here is on hold: {data.blockedBy}.</div>}
        </Card>
        <Card title="Tasks" right={<A to={`/queue?project=${m.projectId}&milestone=${m.id}`}>Open in queue ›</A>}>
          {data.tasks.length === 0 ? <span className="muted small">No tasks in this milestone yet.</span> :
            <table className="t"><tbody><tr><th>Task</th><th>Assignee</th><th>Status</th><th>Reqs</th></tr>
              {data.tasks.map((t: any) => <tr key={t.id}><td>{t.id} {t.title}{t.blockedBy && <div className="small muted">{t.blockedBy}</div>}</td><td>{t.assignee || <span className="muted">{t.role}</span>}</td><td><Status s={t.status} /></td><td className="mono">{t.reqs.join(', ')}</td></tr>)}</tbody></table>}
        </Card>
      </div>
      <div className="col" style={{ width: 340 }}>
        {m.gateName && <Card title={<>◆ Gate · {m.gateName}</>} style={{ borderColor: 'var(--acc)' }}>
          <div className="small muted" style={{ marginBottom: 6 }}>Type: {m.gateKind || 'review'} · <Status s={m.gateStatus} /></div>
          <div className="chk">{m.gateCriteria.map((c: string) => <span key={c}>{m.gateStatus === 'approved' ? '✅' : '⬜'} {c}</span>)}</div>
          <div className="row c wrap" style={{ marginTop: 10, gap: 6 }}>
            {data.rehearsal ? <A to={'/rehearsals/' + data.rehearsal.id} className="btn">Open rehearsal {data.rehearsal.id}</A> : <A to={`/rehearsals/new?project=${m.projectId}&milestone=${m.id}`} className="btn">Schedule rehearsal</A>}
            {m.gateKind === 'review' && m.gateStatus !== 'approved' && <button type="button" className="btn p" onClick={() => run(() => api.gates.set(m.id, { status: 'approved' }), 'Gate approved')}>Approve gate</button>}
          </div>
        </Card>}
        <Card title="Requirements"><div className="row c wrap" style={{ gap: 4 }}>{data.requirements.length ? data.requirements.map((r: string) => <Chip key={r} c="blue">{r}</Chip>) : <span className="muted small">None linked yet.</span>}</div></Card>
        <Card title="History"><div className="col small" style={{ gap: 6 }}>{data.history.slice(0, 10).map((e: any) => <div key={e.id}><span className="muted mono">{day(e.ts)}</span> {e.summary} <A to={'/history/' + e.id}>›</A></div>)}{data.history.length === 0 && <span className="muted">Nothing yet.</span>}</div></Card>
      </div>
    </div>
    {edit && <MilestoneDialog projectId={m.projectId} existing={m} onClose={() => setEdit(false)} />}
  </Shell>;
}

// Project tabs: releases, history, settings ------------------------------------
export function ProjectReleases({ id }: { id: string }) {
  const { data } = useData<any>(() => api.projects.get(id), [id]);
  const [dry, setDry] = useState<any>(null);
  const { run, busy } = useAction();
  const l = data?.project?.live || {};
  return <Shell title="Releases" crumb={<A to={`/projects/${id}`}>{id}</A>} tabs={<ProjectTabs id={id} on="releases" />}>
    <div className="row">
      <Card title="Latest stable" style={{ flex: 1 }}>
        <dl className="kv"><dt>Version</dt><dd>{l.release ? <b>{l.release.tag}</b> : 'No stable release yet'}</dd><dt>Date</dt><dd>{l.release?.date || '—'}</dd><dt>Commits since</dt><dd>{l.release?.commitsSince ?? '—'}</dd>
          <dt>Website</dt><dd>{data?.project?.websiteUrl ? <a href={data.project.websiteUrl} target="_blank" rel="noreferrer">{data.project.websiteUrl}</a> : 'Website not published'}</dd></dl>
      </Card>
      <Card title="Release (Courier, no LLM)" style={{ flex: 1 }}>
        <p className="small">Courier tags the next CalVer version, writes notes from the commits since the last release, creates the forge release and publishes Pages (FR-05, FR-44).</p>
        <div className="row c"><button type="button" className="btn" disabled={busy} onClick={() => run(() => api.git.release(id, { dryRun: true })).then(setDry)}>Preview</button>
          <button type="button" className="btn p" disabled={busy} onClick={() => { if (confirmRelease()) run(() => api.git.release(id, { dryRun: false }), 'Released').then(setDry); }}>Release now</button></div>
        {dry && <div style={{ marginTop: 10 }}><b>{dry.tag}</b><ul className="small">{dry.steps?.map((s: string) => <li key={s}>{s}</li>)}</ul><pre className="term" style={{ borderRadius: 6 }}>{dry.notes}</pre></div>}
      </Card>
    </div>
  </Shell>;
}
function confirmRelease() { return window.confirm('Tag and publish a release now?'); }

export function ProjectHistory({ id }: { id: string }) {
  const { data } = useData<any[]>(() => api.history.list({ project: id, limit: 200 }), [id], (e) => e.projectId === id);
  return <Shell title="History" crumb={<A to={`/projects/${id}`}>{id}</A>} tabs={<ProjectTabs id={id} on="history" />}>
    <Card>{(data || []).map((e) => <EventRow key={e.id} e={e} />)}{data?.length === 0 && <Empty>No history yet.</Empty>}</Card>
  </Shell>;
}

export function EventRow({ e }: { e: any }) {
  return <div className="row c small" style={{ padding: '8px 0', borderBottom: '1px solid var(--line)', gap: 10 }}>
    <span className="muted mono" style={{ width: 92, flex: 'none' }}>{day(e.ts)} {new Date(e.ts).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</span>
    <Av s name={e.actor} role={e.actorKind === 'human' ? 'human' : e.actorKind === 'system' ? 'git' : 'developer'} />
    <div className="grow"><b>{e.summary}</b>{e.why && <div className="muted">Why: {e.why}</div>}<div className="muted xs">{e.type}{e.origin !== 'system' ? ` · via ${e.origin}` : ''}{e.projectId ? ` · ${e.projectId}` : ''}</div></div>
    <A to={'/history/' + e.id}>Open ›</A></div>;
}

export function ProjectSettings({ id }: { id: string }) {
  const { data } = useData<any>(() => api.projects.get(id), [id]);
  const { data: teams } = useData<any[]>(() => api.teams.list(), []);
  const { data: wf } = useData<any>(() => api.git.workflow(id), [id]);
  const { run } = useAction();
  const p = data?.project;
  if (!p) return <Shell title="Settings" crumb={id}><Loading /></Shell>;
  return <Shell title="Settings" crumb={<A to={`/projects/${id}`}>{id}</A>} tabs={<ProjectTabs id={id} on="settings" />}>
    <div className="row">
      <Card title="Project" style={{ flex: 1 }}>
        <div className="col" style={{ gap: 10 }}>
          <div className="form-row"><label>Stage</label><select value={p.stage} onChange={(e) => run(() => api.projects.update(id, { stage: e.target.value }), 'Stage changed')}>{STAGES.map((s) => <option key={s}>{s}</option>)}</select></div>
          <div className="form-row"><label>Team</label><select value={p.teamId} onChange={(e) => run(() => api.projects.update(id, { teamId: e.target.value }), 'Team assigned')}><option value="">(any persona)</option>{(teams || []).map((t) => <option key={t.id} value={t.id}>{t.name}</option>)}</select></div>
          <div className="form-row"><label>Website URL</label><input defaultValue={p.websiteUrl} onBlur={(e) => e.target.value !== p.websiteUrl && run(() => api.projects.update(id, { websiteUrl: e.target.value }), 'Saved')} /></div>
          <button type="button" className="btn d" style={{ alignSelf: 'flex-start' }} onClick={() => { if (window.confirm('Unregister this project? Files are not touched.')) run(() => api.projects.remove(id), 'Removed').then(() => navigate('/')); }}>Remove from go-build-it</button>
        </div>
      </Card>
      <Card title="Git workflow" style={{ flex: 1 }}>
        <p className="small">{wfText(wf)}</p><p className="small muted">From {wf?.source}. The workflow is a global standard; override it per project in <A to="/standards">Standards</A>.</p>
      </Card>
    </div>
  </Shell>;
}

export { MilestoneDialog };
