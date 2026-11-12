import { useEffect, useRef, useState } from 'react';
import { api } from '../api.gen';
import { useData } from '../live';
import { navigate, query } from '../router';
import { ProjectTabs, Shell } from '../shell';
import { A, AgentChat, Card, Chip, DictArea, Editor, Empty, Err, Loading, Proposals, applyHunks, day, kindOf, useAction, type Hunk } from '../ui';

import { usePersonas } from './Team';

// v21–v26 · Project documents (FR-140–FR-150, ADR-014). Six sub-tabs: four HTML entry
// points shown in a sandboxed viewer that follows links, Compare, and Support files.
type DocTab = { id: string; title: string; kind: 'entry' | 'compare' | 'files'; path?: string };
const STATE: Record<string, [string, string]> = { 'accepted': ['grn', 'accepted'], 'drifted': ['amb', 'drifted'], 'not-reviewed': ['red', 'not reviewed'], 'task': ['blue', 'task filed'], 'no-screenshot': ['', 'no screenshot'] };
const docEvent = (e: any) => e.type.startsWith('artifact') || e.type.startsWith('task');

export function Artifacts({ id, tab: tabIn, doc }: { id: string; tab?: string; doc?: string }) {
  const { data: files, error, reload } = useData<any>(() => api.artifacts.files(id), [id], docEvent);
  const { data: cmp } = useData<any>(() => api.compare.list(id), [id], docEvent);
  // docs/gbi.toml decides which sub-tabs exist, their order and their names (FR-151).
  const tabs: DocTab[] = files?.tabs || [];
  const tab = tabIn || (query('path') ? 'files' : tabs[0]?.id || 'files');
  const cur = tabs.find((t) => t.id === tab);
  const unl = files?.unlabelled?.length || 0;
  return <Shell title="Artifacts" crumb={<A to={`/projects/${id}`}>{id}</A>} tabs={<ProjectTabs id={id} on="artifacts" />}
    right={unl > 0 ? <Chip c="amb">{unl} unlabelled</Chip> : undefined}>
    <Err msg={error} />
    <nav className="subtabs" aria-label="Project documents">{tabs.map((t) => <A key={t.id} to={`/projects/${id}/artifacts/${t.id}`} className={tab === t.id ? 'on' : ''}>{t.title}
      {t.kind === 'compare' && cmp?.toReview > 0 && <> <Chip c="red">{cmp.toReview}</Chip></>}{t.kind === 'files' && unl > 0 && <> <Chip c="amb">{unl}</Chip></>}</A>)}</nav>
    {!files ? <Loading /> : tab === 'compare' ? <CompareTab id={id} view={doc || ''} data={cmp} />
      : tab === 'files' ? <SupportFiles id={id} files={files} reload={reload} />
        : !cur && !files.entries?.[tab] ? <Card title="No such tab"><div className="small">This project has no <b>{tab}</b> tab. The sub-tabs come from <span className="mono">docs/gbi.toml</span>: each role in <span className="mono">[entries]</span> gets one, in the order written, or as listed in <span className="mono">tabs = [...]</span>. <A to={`/projects/${id}/artifacts/files`}>Support files ›</A></div></Card>
          : <EntryTab key={tab + ':' + (files.entries?.[tab] || '')} id={id} role={tab} title={cur?.title || tab} doc={doc || ''} files={files} reload={reload} />}
  </Shell>;
}

function relFromRaw(id: string, pathname: string) {
  const pre = `/raw/${id}/`;
  return pathname.startsWith(pre) ? decodeURIComponent(pathname.slice(pre.length)) : '';
}

function EntryTab({ id, role, title, doc, files, reload }: { id: string; role: string; title: string; doc: string; files: any; reload: () => void }) {
  const entries: Record<string, string> = files.entries || {};
  const entry = entries[role];
  const start = doc || entry || '';
  const [src] = useState(() => start ? `/raw/${id}/${start}${location.hash}` : '');
  const [here, setHere] = useState({ path: start, hash: location.hash });
  const [trace, setTrace] = useState(query('trace') === '1');
  const [rev, setRev] = useState('');
  const frame = useRef<HTMLIFrameElement>(null);
  const { run, busy } = useAction();
  const { data: art } = useData<any>(() => (here.path ? api.artifacts.get(id, here.path) : Promise.resolve(null)), [id, here.path]);
  useEffect(() => {
    const on = (e: MessageEvent) => {
      if (!frame.current || e.source !== frame.current.contentWindow || e.data?.gbi !== 'nav') return;
      const u = new URL(e.data.href);
      let rel = relFromRaw(id, u.pathname);
      if (!rel) return;
      const other = Object.entries(entries).find(([r, p]) => p === rel && r !== role);
      const to = `/projects/${id}/artifacts/${other ? other[0] : role}${rel === entries[other ? other[0] : role] ? '' : '/' + rel}${u.hash}`;
      if (other) { navigate(to); return; }
      history.replaceState(null, '', to);
      setHere({ path: rel, hash: u.hash });
    };
    // Editing the anchor in the address bar moves the document there too.
    const onHash = () => { setHere((h) => ({ ...h, hash: location.hash })); frame.current?.contentWindow?.postMessage({ gbi: 'hash', hash: location.hash }, '*'); };
    window.addEventListener('message', on);
    window.addEventListener('hashchange', onHash);
    return () => { window.removeEventListener('message', on); window.removeEventListener('hashchange', onHash); };
  }, [id, role, entries]);
  if (!entry) return <NoEntry id={id} role={role} title={title} files={files} reload={reload} />;
  const label = title;
  const md = (art?.content || '').match(/<gbi-md[^>]*\bsrc="([^"]+)"/);
  const source = here.path.endsWith('.md') ? here.path : md ? joinRel(here.path, md[1]) : here.path;
  const frameSrc = rev ? `/raw/${id}/${here.path}?rev=${rev}` : src;
  return <div className="row">
    <div className="grow">
      <div className="docbar">
        <Chip c="acc">{label}</Chip><span className="mono small">{here.path}{here.hash}</span>
        {art && <span className="small muted">rev {art.rev}{art.revisions?.[0] && <> · {art.revisions[0].author} · {day(art.revisions[0].date)}</>}</span>}
        {role === 'requirements' && <span className="seg"><button type="button" className={trace ? '' : 'on'} onClick={() => setTrace(false)}>Document</button><button type="button" className={trace ? 'on' : ''} onClick={() => setTrace(true)}>Traceability</button></span>}
        <span className="right" />
        {art?.revisions?.length > 1 && <select value={rev} onChange={(e) => setRev(e.target.value)} aria-label="Revision"><option value="">current</option>{art.revisions.map((r: any, k: number) => <option key={r.hash} value={r.hash}>rev {art.revisions.length - k} · {r.hash}</option>)}</select>}
        {role === 'tracker' && <button type="button" className="btn sm p" disabled={busy} onClick={() => run(() => api.artifacts.snapshot(id), 'Snapshot written').then(() => { if (frame.current) frame.current.src = frame.current.src; })}>Write snapshot</button>}
        <A to={`/projects/${id}/edit?path=${encodeURIComponent(source)}`} className="btn sm">✎ Edit source</A>
        <a className="btn sm" href={`/raw/${id}/${here.path}${here.hash}`} target="_blank" rel="noreferrer">Open raw ↗</a>
      </div>
      {trace ? <TraceTable id={id} /> : <div className="docframe"><iframe ref={frame} key={frameSrc} src={frameSrc} title={`${label} · ${here.path}`} sandbox="allow-scripts allow-forms allow-popups allow-popups-to-escape-sandbox" /><span className="sandboxed">sandboxed · no access to gbi</span></div>}
    </div>
    {role === 'requirements' && <ProjectRail id={id} />}
    {role === 'mockups' && <OnThisView id={id} hash={here.hash} />}
  </div>;
}

function joinRel(from: string, link: string) {
  const parts = from.split('/'); parts.pop();
  for (const seg of link.split('/')) { if (seg === '..') parts.pop(); else if (seg !== '.') parts.push(seg); }
  return parts.join('/');
}

function NoEntry({ id, role, title, files, reload }: { id: string; role: string; title: string; files: any; reload: () => void }) {
  const { run, busy } = useAction();
  const html = (files.files || []).filter((f: any) => f.kind === 'html');
  const [pick, setPick] = useState('');
  return <Card title={`No ${title} entry point yet`}>
    <div className="col small" style={{ gap: 10 }}>
      <div>Each tab opens one HTML file you choose. go-build-it never picks one for you. <span className="muted">CLI: <code>gbi artifacts entry {id} {role} docs/&lt;file&gt;.html</code></span></div>
      {html.length > 0 && <div className="row c"><select value={pick} onChange={(e) => setPick(e.target.value)} aria-label="HTML file"><option value="">Choose an HTML file…</option>{html.map((f: any) => <option key={f.path} value={f.path}>{f.path}</option>)}</select>
        <button type="button" className="btn p" disabled={!pick || busy} onClick={() => run(() => api.artifacts.entry(id, role, pick), `The ${role} entry point is ${pick}`).then(reload)}>Set as entry point</button></div>}
      {['requirements', 'mockups', 'plan', 'tracker'].includes(role) && <div className="row c"><button type="button" className="btn" data-tour="docs-init" disabled={busy} onClick={() => run(() => api.artifacts.init(id), 'Created the standard documents').then(reload)}>Create the standard set</button>
        <span className="muted">writes templates for every missing document; never overwrites · <code>gbi artifacts init {id}</code></span></div>}
    </div>
  </Card>;
}

function ProjectRail({ id }: { id: string }) {
  const { data: vals } = useData<any[]>(() => api.values.list(id), [id], (e) => e.type.startsWith('value'));
  const { data: prob } = useData<any>(() => api.problem.get(id), [id], (e) => e.type.startsWith('problem'));
  const [newVal, setNewVal] = useState('');
  const [probEdit, setProbEdit] = useState<string | null>(null);
  const { run } = useAction();
  return <div className="col" style={{ width: 280 }}>
    <Card title={<>Problem definition {prob?.version ? `v${prob.version}` : ''}</>} right={<button type="button" className="lnk" onClick={() => setProbEdit(probEdit === null ? prob?.text || '' : null)}>Refine</button>}>
      {probEdit !== null ? <div className="col" style={{ gap: 6 }}><DictArea value={probEdit} onChange={setProbEdit} rows={4} />
        <button type="button" className="btn p" onClick={() => run(() => api.problem.refine(id, { text: probEdit, source: 'edited by hand' }), 'Saved new version').then(() => setProbEdit(null))}>Save as new version</button></div>
        : <div className="small">{prob?.text || <span className="muted">Not written yet.</span>}{prob?.source && <div className="muted" style={{ marginTop: 6 }}>From: {prob.source} · {day(prob.createdAt)}</div>}</div>}
    </Card>
    <Card title="Values">
      <div className="col small" style={{ gap: 8 }}>
        {(vals || []).map((v, k) => v.status === 'proposed' ? <div key={v.id} style={{ background: 'var(--acc-s)', padding: '6px 8px', borderRadius: 6 }}><b>Proposed:</b> "{v.text}" <span className="muted">{v.source}</span><br />
          <button type="button" className="lnk" onClick={() => run(() => api.values.set(String(v.id), { status: 'accepted' }), 'Value accepted')}>Accept</button> · <button type="button" className="lnk" onClick={() => { const t = window.prompt('Edit value', v.text); if (t) run(() => api.values.set(String(v.id), { status: 'accepted', text: t }), 'Value accepted'); }}>Edit</button> · <button type="button" className="lnk" onClick={() => run(() => api.values.set(String(v.id), { status: 'dismissed' }), 'Dismissed')}>Dismiss</button></div>
          : <div key={v.id}><b>{k + 1}. {v.text}</b> {v.source && <span className="muted">← {v.source}</span>}</div>)}
        {!vals?.length && <span className="muted">No values yet. They are proposed from your answers and choices.</span>}
        <div className="row c" style={{ gap: 6 }}><input className="grow" placeholder="Add a value…" value={newVal} onChange={(e) => setNewVal(e.target.value)} />
          <button type="button" className="btn" disabled={!newVal} onClick={() => run(() => api.values.add(id, { text: newVal, source: 'added by hand' })).then(() => setNewVal(''))}>Add</button></div>
      </div>
    </Card>
  </div>;
}

function OnThisView({ id, hash }: { id: string; hash: string }) {
  const { data: tr } = useData<any>(() => api.trace.list(id), [id], docEvent);
  const vid = decodeURIComponent(hash.replace(/^#/, ''));
  const v = (tr?.views || []).find((x: any) => x.id === vid);
  const tasks = (tr?.rows || []).filter((r: any) => v?.reqs?.includes(r.id)).flatMap((r: any) => r.tasks);
  return <Card title="On this view" style={{ width: 270 }}>
    {!v ? <div className="small muted">Open a view (an anchor such as <span className="mono">#v01</span> on a <span className="mono">data-view</span> section) to see its requirements, tasks and real screen.</div>
      : <div className="col small" style={{ gap: 6 }}>
        <div><b>{viewName(v)}</b>{v.route && <> · <span className="mono">{v.route}</span></>}</div>
        <div className="muted">Requirements</div><div>{v.reqs.length ? v.reqs.map((r: string) => <A key={r} to={`/projects/${id}/artifacts/requirements#${r}`}><span className="idtag">{r}</span> </A>) : '—'}</div>
        <div className="muted">Tasks</div><div>{tasks.length ? [...new Set(tasks)].join(', ') : '—'}</div>
        <div className="muted">Real screen</div><div><Chip c={STATE[v.state]?.[0]}>{STATE[v.state]?.[1]}</Chip> <A to={`/projects/${id}/artifacts/compare/${v.id}`}>Compare ›</A></div>
      </div>}
  </Card>;
}

function TraceTable({ id }: { id: string }) {
  const [gaps, setGaps] = useState(false);
  const { data: tr } = useData<any>(() => api.trace.list(id, gaps ? { gaps: true } : undefined), [id, gaps], docEvent);
  if (!tr) return <Loading />;
  return <Card title={<>{tr.rows.length} requirements · {tr.gaps} gaps</>} right={<span className="seg"><button type="button" className={gaps ? '' : 'on'} onClick={() => setGaps(false)}>All</button><button type="button" className={gaps ? 'on' : ''} onClick={() => setGaps(true)}>Gaps only</button></span>}>
    <table className="t small trace"><thead><tr><th>Requirement</th><th>Mockup views</th><th>Tasks</th><th>Done</th><th>Real screen</th><th>Gaps</th></tr></thead><tbody>
      {tr.rows.map((r: any) => <tr key={r.id}><td><A to={`/projects/${id}/artifacts/requirements#${r.id}`}><span className="idtag">{r.id}</span></A> {r.text}</td>
        <td>{r.views.map((v: string) => <A key={v} to={`/projects/${id}/artifacts/compare/${v}`}>{v} </A>)}{!r.views.length && <span className="muted">—</span>}</td>
        <td>{r.tasks.join(', ') || <span className="muted">—</span>}</td><td>{r.tasks.length ? `${r.done}/${r.tasks.length}` : '—'}</td>
        <td>{r.screens.map((s: string) => { const [v, st] = s.split(':'); return <Chip key={s} c={STATE[st]?.[0]}>{v} {STATE[st]?.[1]}</Chip>; })}</td>
        <td>{r.gaps.map((g: string) => <span key={g} className="gap">{g}</span>)}</td></tr>)}
      {!tr.rows.length && <tr><td colSpan={6} className="muted">{gaps ? 'No gaps.' : 'No requirements found. Requirements are "**FR-nn**:" lines in the requirements entry or its Markdown.'}</td></tr>}
    </tbody></table>
    {tr.viewGaps?.length > 0 && <div className="small" style={{ marginTop: 8 }}>{tr.viewGaps.map((g: string) => <div key={g}><span className="gap">{g}</span></div>)}</div>}
    <div className="small muted" style={{ marginTop: 8 }}>Read from the IDs only. CLI: <code>gbi trace list {id} --gaps</code></div>
  </Card>;
}

const viewName = (v: any) => (v.title && v.title.startsWith(v.id) ? v.title : `${v.id}${v.title ? ' · ' + v.title : ''}`);

function CompareTab({ id, view, data }: { id: string; view: string; data: any }) {
  const views: any[] = data?.views || [];
  const sel = views.find((v) => v.id === view) || views.find((v) => v.state === 'drifted' || v.state === 'not-reviewed') || views[0];
  const [mode, setMode] = useState<'side' | 'overlay'>(query('mode') === 'overlay' ? 'overlay' : 'side');
  const [split, setSplit] = useState(50);
  const [note, setNote] = useState('');
  const { run, busy } = useAction();
  useEffect(() => setNote(''), [sel?.id]);
  if (!data) return <Loading />;
  if (!views.length) return <Card title="Compare"><div className="small">No mockup views yet. A view is an element with <code>data-view="v01"</code> in an HTML file labelled <span className="lbl">mockup</span>, and its screenshot is an image labelled <span className="lbl">screenshot</span> named after it (<code>v01.png</code>).</div></Card>;
  const mock = `/raw/${id}/${sel.file}?gbi-view=${encodeURIComponent(sel.id)}`;
  const shot = sel.screenshot ? `/raw/${id}/${sel.screenshot}?v=${sel.decision?.sha256 || ''}${Date.now() % 1}` : '';
  return <div className="row">
    <Card title={<>Views <span className="muted">{data.counts?.accepted || 0}/{views.length} ok</span></>} style={{ width: 210, padding: 8 }}>
      <div className="col" style={{ gap: 2 }}>{views.map((v) => <A key={v.id} to={`/projects/${id}/artifacts/compare/${v.id}`} className={'vi' + (v.id === sel.id ? ' on' : '')}><span className={'dot ' + (STATE[v.state]?.[0] || '')} /> {viewName(v)}</A>)}</div>
      <div className="xs muted" style={{ marginTop: 8 }}>{Object.entries(STATE).map(([k, [c, l]]) => <span key={k} style={{ marginRight: 8 }}><span className={'dot ' + c} /> {l}</span>)}</div>
    </Card>
    <div className="grow">
      <div className="docbar"><b>{viewName(sel)}</b> {sel.route && <span className="mono small muted">{sel.route}</span>}
        <span className="seg"><button type="button" className={mode === 'side' ? 'on' : ''} onClick={() => setMode('side')}>Side by side</button><button type="button" className={mode === 'overlay' ? 'on' : ''} onClick={() => setMode('overlay')}>Overlay</button></span>
        <span className="right" /><Chip c={STATE[sel.state]?.[0]}>{STATE[sel.state]?.[1]}</Chip></div>
      <div className="compare">
        {mode === 'side' ? <>
          <figure><figcaption>Mockup · <span className="mono">{sel.file}#{sel.id}</span></figcaption><iframe src={mock} title={`Mockup ${sel.id}`} sandbox="allow-scripts" /></figure>
          <figure><figcaption>Real app · <span className="mono">{sel.screenshot || 'no screenshot'}</span></figcaption>{shot ? <img src={shot} alt={`Screenshot of ${sel.id}`} /> : <div className="noshot">No screenshot named {sel.id}.png labelled screenshot yet.</div>}</figure>
        </> : <figure><figcaption>Mockup left of the line · real app right · drag the slider</figcaption>
          <div className="overlay-cmp"><iframe src={mock} title={`Mockup ${sel.id}`} sandbox="allow-scripts" />{shot && <img src={shot} alt={`Screenshot of ${sel.id}`} style={{ clipPath: `inset(0 0 0 ${split}%)` }} />}<i style={{ left: `${split}%` }} /></div>
          <input type="range" min={0} max={100} value={split} onChange={(e) => setSplit(+e.target.value)} aria-label="Overlay position" style={{ width: '100%' }} /></figure>}
      </div>
    </div>
    <div className="col" style={{ width: 280 }}>
      <Card title="Decision">
        <div className="col small" style={{ gap: 8 }}>
          {sel.decision && <div className="muted">{sel.decision.decision === 'task' ? <>Task {sel.decision.task} filed</> : 'Accepted'} by {sel.decision.by} · {day(sel.decision.at)}{sel.state === 'drifted' && <> · <b>the screenshot changed since</b></>}{sel.decision.note && <div>“{sel.decision.note}”</div>}</div>}
          {sel.screenshot ? <><DictArea value={note} onChange={setNote} rows={3} placeholder="What is different, or why it is fine" />
            <div className="row c" style={{ gap: 6 }}><button type="button" className="btn" disabled={busy} onClick={() => run(() => api.compare.accept(id, sel.id, { note }), `Accepted ${sel.id}`)}>✓ Accept the real screen</button>
              <button type="button" className="btn p" disabled={busy || !note.trim()} onClick={() => run(() => api.compare.task(id, sel.id, { note }), 'Task filed')}>File a task</button></div>
            <div className="muted">Saved in <span className="mono">docs/compare.json</span> with the screenshot's SHA-256; the view comes back when the screenshot changes.</div></>
            : <div className="muted">Add a screenshot named <span className="mono">{sel.id}.png</span> to a folder labelled screenshot.</div>}
        </div>
      </Card>
      <Card title="Covers"><div className="small">{sel.reqs?.length ? sel.reqs.map((r: string) => <A key={r} to={`/projects/${id}/artifacts/requirements#${r}`}><span className="idtag">{r}</span> </A>) : <span className="muted">no data-req</span>}</div>
        <div className="small muted" style={{ marginTop: 6 }}>CLI: <code>gbi compare list {id}</code></div></Card>
    </div>
  </div>;
}

function SupportFiles({ id, files, reload }: { id: string; files: any; reload: () => void }) {
  const [filter, setFilter] = useState(query('label') || 'all');
  const [edit, setEdit] = useState<string | null>(null);
  const [pick, setPick] = useState<string[]>([]);
  const [folder, setFolder] = useState('');
  const [preview, setPreview] = useState(query('path'));
  const { data: check } = useData<any>(() => api.artifacts.check(id), [id], docEvent);
  const { run, busy } = useAction();
  const all: any[] = files.files || [];
  const labels: string[] = files.allLabels || [];
  const unl: any[] = files.unlabelled || [];
  const shown = filter === 'all' ? all.filter((f) => f.labels.length || f.kept) : filter === 'unlabelled' ? unl : all.filter((f) => f.labels.includes(filter));
  const used = labels.filter((l) => all.some((f) => f.labels.includes(l)));
  const startEdit = (f: any) => { setEdit(f.path); setPick(f.from ? [] : f.labels); };
  const save = (path: string, isNew: boolean) => run(() => (isNew ? api.artifacts.add : api.artifacts.label)(id, path, { label: pick.join(',') }), `Labelled ${path}`).then((r) => { if (r) { setEdit(null); reload(); } });
  const row = (f: any) => <tr key={f.path} className={preview === f.path ? 'sel' : ''}>
    <td><button type="button" className="lnk" onClick={() => setPreview(f.path)}>{f.path}</button></td>
    <td>{f.entry && <span className="lbl entry">entry · {f.entry}</span>} {f.labels.map((l: string) => <span key={l} className="lbl">{l}</span>)}{f.from && <span className="muted xs"> from {f.from}</span>}{f.kept && <span className="muted xs">kept by gbi</span>}{!f.labels.length && !f.kept && <span className="lbl none">unlabelled</span>}</td>
    <td style={{ textAlign: 'right', whiteSpace: 'nowrap' }}>{!f.kept && (edit === f.path ? <span className="labelpick">{labels.map((l) => <label key={l}><input type="checkbox" checked={pick.includes(l)} onChange={(e) => setPick(e.target.checked ? [...pick, l] : pick.filter((x) => x !== l))} />{l}</label>)}
      <button type="button" className="btn sm p" disabled={busy || !pick.length} onClick={() => save(f.path, !f.labels.length)}>Save</button> <button type="button" className="btn sm" onClick={() => setEdit(null)}>Cancel</button></span>
      : <button type="button" className="btn sm" onClick={() => startEdit(f)}>Label…</button>)}
      {f.kind === 'html' && !f.entry && edit !== f.path && <select className="sm" value="" onChange={(e) => e.target.value && run(() => api.artifacts.entry(id, e.target.value, f.path), `${f.path} is the ${e.target.value} entry point`).then(reload)} aria-label={`Set ${f.path} as entry point`}><option value="">Set as entry point…</option>{(files.roles || []).map((r: string) => <option key={r} value={r}>{r}</option>)}</select>}</td></tr>;
  const pv = all.find((f) => f.path === preview);
  return <div className="row">
    <div className="col grow">
      <Card title={<>docs/ <span className="muted">{all.length} files</span></>} right={<span className="seg">
        {[['all', 'Labelled'], ['unlabelled', `Unlabelled ${unl.length}`], ...used.map((l) => [l, l])].map(([k, l]) => <button type="button" key={k} className={filter === k ? 'on' : ''} onClick={() => setFilter(k)}>{l}</button>)}</span>}>
        {filter === 'all' && unl.length > 0 && <div className="note" style={{ marginBottom: 8 }}><b>{unl.length} unlabelled</b>: added outside gbi. Give each a label; gbi never guesses. <button type="button" className="lnk" onClick={() => setFilter('unlabelled')}>Show them</button></div>}
        <table className="t small"><thead><tr><th>File</th><th>Labels</th><th></th></tr></thead><tbody>{shown.map(row)}
          {!shown.length && <tr><td colSpan={3} className="muted">{filter === 'unlabelled' ? 'Every file has a label.' : 'Nothing here yet. Run gbi artifacts init, or label files with gbi artifacts add.'}</td></tr>}</tbody></table>
      </Card>
      {pv && <Card title={<span className="mono">{pv.path}</span>} right={<span className="small"><A to={`/projects/${id}/edit?path=${encodeURIComponent(pv.path)}`} className="btn sm">✎ Edit</A> <button type="button" className="btn sm" onClick={() => setPreview('')}>Close</button></span>}>
        {pv.kind === 'image' ? <img src={`/raw/${id}/${pv.path}`} alt={pv.path} style={{ maxWidth: '100%' }} /> : <div className="docframe short"><iframe src={`/raw/${id}/${pv.path}${pv.kind === 'markdown' ? '?gbi=md' : ''}`} title={pv.path} sandbox="allow-scripts allow-popups" /></div>}
      </Card>}
    </div>
    <div className="col" style={{ width: 320 }}>
      <Card title="Entry points"><table className="t small"><tbody>{(files.roles || []).map((r: string) => <tr key={r}><td>{r}</td><td className="mono">{files.entries?.[r] ? <A to={`/projects/${id}/artifacts/${r}`}>{files.entries[r].replace(/^docs\//, '')}</A> : <span className="muted">not set</span>}</td></tr>)}</tbody></table></Card>
      <Card title={<>Check {check && <Chip c={check.ok ? 'grn' : 'amb'}>{check.ok ? 'all good' : `${check.problems.length} problems`}</Chip>}</>}>
        <div className="col small" style={{ gap: 4 }}>{(check?.problems || []).slice(0, 12).map((p: any, k: number) => <div key={k}>⚠ {p.detail}</div>)}{check?.problems?.length > 12 && <div className="muted">…and {check.problems.length - 12} more: <code>gbi artifacts check {id}</code></div>}</div></Card>
      <Card title="Label a folder"><div className="col small" style={{ gap: 6 }}>
        <input placeholder="docs/screenshots/" value={folder} onChange={(e) => setFolder(e.target.value)} aria-label="Folder" />
        <div className="labelpick">{labels.map((l) => <label key={l}><input type="checkbox" checked={pick.includes(l) && edit === null} onChange={(e) => { setEdit(null); setPick(e.target.checked ? [...pick, l] : pick.filter((x) => x !== l)); }} />{l}</label>)}</div>
        <button type="button" className="btn sm" disabled={busy || !folder || !pick.length} onClick={() => run(() => api.artifacts.label(id, folder.endsWith('/') ? folder : folder + '/', { label: pick.join(',') }), `Labelled ${folder}`).then((r) => { if (r) { setFolder(''); setPick([]); reload(); } })}>Label folder</button>
        <div className="muted">A folder's label covers every file in it.</div></div></Card>
      <Card title="From the CLI"><pre className="evi">{`gbi artifacts add ${id} docs/<file> --label <label>
gbi artifacts label ${id} docs/<folder>/ --label mockup
gbi artifacts entry ${id} mockups docs/mockups/index.html
gbi artifacts check ${id}`}</pre><div className="xs muted">A label is required. Each change is committed in docs/gbi.toml.</div></Card>
    </div>
  </div>;
}

// v03b · Artifact editor: chat with an agent or edit directly (FR-45a)
export function ArtifactEditor({ id }: { id: string }) {
  const path = query('path');
  const personas = usePersonas(id);
  const [persona, setPersona] = useState('');
  const [text, setText] = useState<string | null>(null);
  const [base, setBase] = useState('');
  const [rev, setRev] = useState(0);
  const [mode, setMode] = useState<'wysiwyg' | 'markdown' | 'split'>('wysiwyg');
  const [prop, setProp] = useState<{ doc: string; hunks: Hunk[]; base: string } | null>(null);
  const [accepted, setAccepted] = useState<Set<number>>(new Set());
  const [msg, setMsg] = useState('');
  const [err, setErr] = useState('');
  const { run, busy } = useAction();
  useEffect(() => {
    api.artifacts.get(id, path).then((a) => { setText(a.content); setBase(a.content); setRev(a.rev); }).catch(() => { setText('# New document\n'); setBase(''); setRev(0); });
  }, [id, path]);
  useEffect(() => { if (!persona && personas.length) setPersona((personas.find((p) => p.role === 'architect') || personas[0]).id); }, [personas, persona]);
  if (!path) return <Shell title="Editor"><Err msg="No document path given." /></Shell>;
  if (text === null) return <Shell title="Editor"><Loading /></Shell>;
  const dirty = text !== base;
  const applyProposals = async () => { if (!prop) return; const next = await applyHunks(prop.base, prop.hunks, accepted); setText(next); setProp(null); setAccepted(new Set()); };
  const save = () => run(() => api.artifacts.save(id, path, { content: text, message: msg || undefined }), `Saved as Rev ${rev + 1}`).then((r) => { if (r) { setBase(text); setRev(r.rev); navigate(`/projects/${id}/artifacts/files?path=${encodeURIComponent(path)}`); } });
  return <Shell title={<>Editing {path.split('/').pop()}</>} crumb={<><A to={`/projects/${id}`}>{id}</A> / <A to={`/projects/${id}/artifacts/files?path=${encodeURIComponent(path)}`}>Artifacts</A></>}
    right={<>{prop && <Chip c="amb">{prop.hunks.length} proposal{prop.hunks.length > 1 ? 's' : ''}</Chip>}{dirty && <Chip c="amb">unsaved</Chip>}</>}>
    <Err msg={err} />
    <div className="row">
      <div className="col grow">
        {prop && <><Proposals hunks={prop.hunks} accepted={accepted} setAccepted={setAccepted} />
          <div className="row c"><span className="small muted">Accepted changes are applied to the document below; you can keep editing before saving.</span><span className="right" />
            <button type="button" className="btn" onClick={() => { setProp(null); setAccepted(new Set()); }}>Discard proposals</button>
            <button type="button" className="btn p" onClick={applyProposals}>Apply {accepted.size} accepted</button></div></>}
        <Editor value={text} onChange={(v) => { setText(v); setErr(''); }} mode={mode} onMode={setMode} toolbarRight={<>Based on Rev {rev} · <span className="mono">{path}</span></>} />
        <div className="row c">
          <input className="grow" placeholder={`Commit message (default: docs: update ${path} (rev ${rev + 1}))`} value={msg} onChange={(e) => setMsg(e.target.value)} />
          <A to={`/projects/${id}/artifacts/files?path=${encodeURIComponent(path)}`} className="btn">Discard</A>
          <button type="button" className="btn p" disabled={busy || !dirty} onClick={save}>Save as Rev {rev + 1}</button>
        </div>
        <span className="small muted">Saving commits <span className="mono">{path}</span> through Courier and pushes it when the project has a remote (FR-46). CLI: <code>gbi artifacts save {id} {path} --content …</code></span>
      </div>
      <div style={{ width: 340 }}>
        <AgentChat title="Edit with" personas={personas} persona={persona} setPersona={setPersona}
          suggestions={['Tighten wording', 'Check against values', 'Find conflicts']}
          send={(m, h) => api.artifacts.chat(id, { path, persona, message: m, content: text, history: h })}
          onProposal={(doc, hunks) => { setProp({ doc, hunks, base: text }); setAccepted(new Set(hunks.map((_, k) => k))); }} />
      </div>
    </div>
  </Shell>;
}

export function attachmentsFor(projectId: string, atts: any[]) {
  return (atts || []).map((a) => ({ name: a.name, kind: a.kind || kindOf(a.path || a.url || a.name), url: a.url || (a.path ? `/raw/${projectId}/${a.path}` : undefined), body: a.body }));
}

export { usePersonas, Empty };
