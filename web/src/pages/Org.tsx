import { useEffect, useState } from 'react';
import { api } from '../api.gen';
import { useData } from '../live';
import { navigate, query } from '../router';
import { Shell } from '../shell';
import { A, AgentChat, Av, Card, Chip, Editor, Empty, Err, Loading, Markdown, Proposals, Status, Toggle, ago, applyHunks, day, useAction, type Hunk } from '../ui';
import { EventRow } from './Projects';
import { usePersonas } from './Team';

// v11 · Global standards with agent chat (FR-50–FR-54, FR-91)
export function Standards({ sid }: { sid?: string }) {
  const { data: list } = useData<any[]>(() => api.standards.list(), [], (e) => e.type.startsWith('standard'));
  const { data: projects } = useData<any[]>(() => api.projects.list(), []);
  const sel = sid || list?.[0]?.id || '';
  const cur = list?.find((s) => s.id === sel);
  const personas = usePersonas();
  const [persona, setPersona] = useState('');
  const [body, setBody] = useState('');
  const [title, setTitle] = useState('');
  const [mode, setMode] = useState<'wysiwyg' | 'markdown' | 'split'>('wysiwyg');
  const [prop, setProp] = useState<{ hunks: Hunk[]; base: string } | null>(null);
  const [acc, setAcc] = useState<Set<number>>(new Set());
  const [preview, setPreview] = useState('');
  const { run, busy } = useAction();
  useEffect(() => { if (cur) { setBody(cur.body); setTitle(cur.title); } else if (sid === 'new') { setBody('# New standard\n\n- Rule\n\n## Why\n'); setTitle(''); } }, [cur?.id, cur?.rev, sid]);
  useEffect(() => { if (!persona && personas.length) setPersona((personas.find((p) => p.role === 'reviewer') || personas[0]).id); }, [personas, persona]);
  const newId = sid === 'new';
  return <Shell title="Global standards">
    <div className="row">
      <Card title="Standards" style={{ width: 210, padding: 10 }}>
        <div className="col small" style={{ gap: 6 }}>{(list || []).map((s) => <A key={s.id} to={'/standards/' + s.id}>{s.id === sel ? <b>▸ {s.title}</b> : s.title}{s.overrides.length > 0 && <Chip c="amb">{s.overrides.length}</Chip>}</A>)}</div>
        {!list?.length && <span className="muted small">No standards yet. Put Markdown files in ~/.go-build-it/standards to seed them, or add one.</span>}
        <A to="/standards/new" className="btn" >+ New standard</A>
      </Card>
      {(cur || newId) ? <div className="col grow">
        {newId && <input placeholder="Title, e.g. Backend stack" value={title} onChange={(e) => setTitle(e.target.value)} style={{ fontWeight: 700 }} />}
        {prop && <><Proposals hunks={prop.hunks} accepted={acc} setAccepted={setAcc} /><div className="row c"><span className="right" /><button type="button" className="btn" onClick={() => setProp(null)}>Discard</button>
          <button type="button" className="btn p" onClick={async () => { setBody(await applyHunks(prop.base, prop.hunks, acc)); setProp(null); }}>Apply {acc.size} accepted</button></div></>}
        <Editor value={body} onChange={setBody} mode={mode} onMode={setMode} toolbarRight={cur ? <><b>{cur.title}</b> · rev {cur.rev} · edited {day(cur.updatedAt)}</> : 'new'} />
        {cur && <div className="grid" style={{ gridTemplateColumns: 'repeat(3, minmax(0,1fr))', gap: 10 }}>
          <Card title="Overrides" style={{ padding: 10 }}>{cur.overrides.map((o: any) => <div key={o.projectId} className="small"><b>{o.projectId}</b>: {o.reason}</div>)}{cur.overrides.length === 0 && <span className="small muted">None</span>}
            <div><button type="button" className="lnk small" onClick={() => { const p = window.prompt('Project id to override for', projects?.[0]?.id || ''); if (!p) return; const reason = window.prompt('Reason (required)'); if (!reason) return; run(() => api.standards.override(cur.id, { project: p, body, reason }), 'Override recorded'); }}>+ Override for a project</button></div></Card>
          <Card title="Delivered as" style={{ padding: 10 }}><div className="small mono">AGENTS.md · CLAUDE.md · GEMINI.md</div><div className="small muted">Regenerated into every session's worktree (FR-53). {projects?.[0] && <button type="button" className="lnk" onClick={() => api.standards.preview(projects[0].id).then((r) => setPreview(r.content))}>preview</button>}</div></Card>
          <Card title="Compliance" style={{ padding: 10 }}><div className="small">Lens checks every review against this standard (FR-54).</div></Card>
        </div>}
        <div className="row c">{cur && <button type="button" className="btn d" onClick={() => { if (window.confirm('Remove this standard?')) run(() => api.standards.remove(cur.id), 'Removed').then(() => navigate('/standards')); }}>Remove</button>}
          <span className="right" /><button type="button" className="btn p" disabled={busy || (cur ? body === cur.body : !title)} onClick={() => {
            const sId = cur ? cur.id : title.toLowerCase().replace(/\W+/g, '-');
            run(() => api.standards.save(sId, { title: cur ? cur.title : title, body, why: 'edited in the standards editor' }), 'Saved and regenerated').then((r) => r && navigate('/standards/' + r.id));
          }}>{cur ? `Save rev ${cur.rev + 1} & regenerate` : 'Create standard'}</button></div>
        {preview && <Card title="Generated instruction file" right={<button type="button" className="lnk" onClick={() => setPreview('')}>close</button>}><pre className="mono" style={{ whiteSpace: 'pre-wrap', maxHeight: 360, overflow: 'auto' }}>{preview}</pre></Card>}
      </div> : <Empty>Select a standard.</Empty>}
      {cur && <div style={{ width: 290 }}><AgentChat title="Edit with" personas={personas} persona={persona} setPersona={setPersona}
        send={(m, h) => api.standards.chat(cur.id, { persona, message: m, content: body, history: h })}
        onProposal={(_d, hunks) => { setProp({ hunks, base: body }); setAcc(new Set(hunks.map((_, k) => k))); }} /></div>}
    </div>
  </Shell>;
}

// v12 · History & insights (NFR-10–NFR-13)
export function History() {
  const [q, setQ] = useState(query('q'));
  const [type, setType] = useState(query('type'));
  const [project, setProject] = useState(query('project'));
  const { data, error } = useData<any[]>(() => api.history.list({ q, type, project, limit: 200 }), [q, type, project], () => true);
  const { data: m } = useData<any>(() => api.metrics.get({ project, days: 7 }), [project]);
  const { data: props } = useData<any[]>(() => api.proposals.list(), [], (e) => e.type.startsWith('proposal'));
  const { data: cmp } = useData<any>(() => api.metrics.compare({ days: 30 }), []);
  const { data: projects } = useData<any[]>(() => api.projects.list(), []);
  const { run } = useAction();
  return <Shell title="History">
    <div className="grid" style={{ gridTemplateColumns: 'repeat(6, minmax(0,1fr))' }}>
      {[['Compactions / task', m?.compactionsPerTask?.toFixed(1)], ['Rework rate', m ? Math.round(m.reworkRate * 100) + '%' : ''], ['Answer time', m ? m.medianAnswerHours.toFixed(1) + 'h' : ''], ['Time / task', m ? Math.round(m.medianTaskMinutes) + 'm' : ''], ['Rate-limit pauses', m?.rateLimitPauses], ['Sessions', m ? `${m.sessionsRun} (${m.sessionsFailed} failed)` : '']].map(([l, v]) =>
        <div key={l as string} className="tile"><div className="lbl">{l}</div><div className="big">{v ?? '…'}</div><div className="small muted">last 7 days</div></div>)}
    </div>
    <div className="row">
      <Card className="grow" title="Event log" right={<span className="row c" style={{ gap: 6 }}>
        <input placeholder="Search what / why…" value={q} onChange={(e) => setQ(e.target.value)} style={{ width: 180 }} />
        <select value={type} onChange={(e) => setType(e.target.value)}><option value="">All types</option>{['task', 'session', 'question', 'review', 'rehearsal', 'options', 'git', 'release', 'value', 'standard', 'member', 'settings', 'chat'].map((t) => <option key={t}>{t}</option>)}</select>
        <select value={project} onChange={(e) => setProject(e.target.value)}><option value="">All projects</option>{(projects || []).map((p) => <option key={p.id}>{p.id}</option>)}</select></span>}>
        <Err msg={error} />{(data || []).map((e) => <EventRow key={e.id} e={e} />)}{data?.length === 0 && <Empty>No matching events.</Empty>}
        <div className="small muted" style={{ marginTop: 8 }}>Export for agents: <code>gbi history list --json</code> · metrics: <code>gbi metrics get --json</code></div>
      </Card>
      <div className="col" style={{ width: 360 }}>
      {cmp && <Card title={<>Handoffs vs compaction <Chip c="blue">experimental</Chip></>}>
        <table className="t small"><tbody><tr><th>Tasks</th><th>Done</th><th>Rework</th><th>Retries</th></tr>
          {cmp.groups.map((g: any) => <tr key={g.name}><td>{g.name}</td><td>{g.tasks}</td><td>{Math.round(g.reworkRate * 100)}%</td><td>{g.avgRetries.toFixed(1)}</td></tr>)}</tbody></table>
        <div className="small" style={{ marginTop: 6 }}><b>{cmp.verdict}</b></div><div className="xs muted">{cmp.criteria}</div></Card>}
      <Card title="Improvement proposals">
        <div className="col small" style={{ gap: 10 }}>{(props || []).map((p) => <div key={p.id} className="tile"><b>{p.id} · {p.title}</b> <Status s={p.status} /><div className="muted">{p.body}</div><div className="xs muted">Evidence: {p.evidence}</div>
          <div style={{ marginTop: 6 }}><Chip>by {p.author}</Chip> {p.status === 'open' && <><button type="button" className="lnk" onClick={() => run(() => api.proposals.decide(p.id, { decision: 'accepted' }), 'Accepted: task created')}>Accept</button> · <button type="button" className="lnk" onClick={() => run(() => api.proposals.decide(p.id, { decision: 'rejected' }), 'Rejected')}>Reject</button></>}</div></div>)}
          {props?.length === 0 && <span className="muted">Agents propose improvements from these metrics with <code>gbi proposals create</code>. Accepted proposals become tasks that go through normal review (NFR-13).</span>}</div>
      </Card>
      </div>
    </div>
  </Shell>;
}

// v12b · Event detail (NFR-14)
export function EventDetail({ eid }: { eid: string }) {
  const { data, error } = useData<any>(() => api.history.get(eid), [eid]);
  if (error) return <Shell title="Event"><Err msg={error} /></Shell>;
  if (!data) return <Shell title="Event"><Loading /></Shell>;
  const e = data.event;
  const d = e.data || {};
  return <Shell title={<>E-{e.id} · {e.summary}</>} crumb={<A to="/history">History</A>} right={<>{e.projectId && <Chip>{e.projectId}</Chip>}<Chip c={e.actorKind === 'human' ? 'acc' : ''}>{e.actorKind} action</Chip></>}>
    <div className="row">
      <div className="col grow">
        <Card><dl className="kv"><dt>Who</dt><dd><Av s name={e.actor} role={e.actorKind === 'human' ? 'human' : 'developer'} /> {e.actor} · via {e.origin}</dd><dt>When</dt><dd>{new Date(e.ts).toLocaleString()}</dd>
          <dt>What</dt><dd>{e.summary}</dd><dt>Why</dt><dd style={{ whiteSpace: 'pre-wrap' }}>{e.why || '—'}</dd><dt>Type</dt><dd className="mono">{e.type} · {e.subject}</dd></dl></Card>
        {data.optionSet && <Card title="What was shown"><div className="grid" style={{ gridTemplateColumns: `repeat(${data.optionSet.options.length}, 1fr)`, gap: 10 }}>
          {data.optionSet.options.map((o: any) => { const chosen = data.optionSet.choice.filter((c: any) => c.option === o.key); return <div key={o.key} className="tile" style={chosen.length ? { outline: '2px solid var(--acc)' } : undefined}><b>{o.key} · {o.title}</b><div className="small muted">{o.summary}</div><div className="small">{chosen.length ? '✓ ' + chosen.map((c: any) => c.aspect || 'whole option').join(', ') : 'not chosen'}</div></div>; })}</div>
          <p style={{ fontStyle: 'italic' }}>"{data.optionSet.reason}"</p></Card>}
        {data.question && <Card title="Question"><b>{data.question.title}</b><p className="small">{data.question.why}</p><p className="small">Answer ({data.question.answeredBy}): {data.question.choice && `chose ${data.question.choice}. `}{data.question.answer}</p></Card>}
        {data.review && <Card title="Review"><p className="small">{data.review.assessment?.summary}</p><p className="small">Verdict: {data.review.verdict || 'pending'} {data.review.comment}</p><A to={'/reviews/' + data.review.id}>Open review</A></Card>}
        {data.rehearsal && <Card title="Rehearsal"><b>{data.rehearsal.title}</b><p className="small">{data.rehearsal.findings.length} findings · {data.rehearsal.status}</p><A to={'/rehearsals/' + data.rehearsal.id}>Open rehearsal</A></Card>}
        {data.session && <Card title="Session" right={<A to={'/sessions/' + data.session.id}>Open terminal view</A>}><div className="small">{data.session.memberId} · {data.session.harness} · {data.session.status} · {Math.round(data.session.fraction * 100)}% context</div>
          <div className="out" style={{ maxHeight: 260, marginTop: 8 }}>{(data.output || []).slice(-40).map((l: any) => <div key={l.seq} className={'k-' + l.kind}>{l.text}</div>)}</div></Card>}
        {Object.keys(d).length > 0 && <details><summary className="small muted">Raw data</summary><pre className="mono" style={{ whiteSpace: 'pre-wrap' }}>{JSON.stringify(d, null, 2)}</pre></details>}
      </div>
      <div className="col" style={{ width: 340 }}>
        <Card title="What happened as a result"><div className="col small" style={{ gap: 6 }}>{(data.consequences || []).slice(0, 12).map((x: any) => <A key={x.id} to={'/history/' + x.id}><Chip>{x.type.split('.')[0]}</Chip> {x.summary}</A>)}{!(data.consequences || []).length && <span className="muted">Nothing followed within 10 minutes.</span>}</div></Card>
        <Card title={<>Related to {e.subject}</>}><div className="col small" style={{ gap: 6 }}>{(data.related || []).filter((x: any) => x.id !== e.id).slice(0, 12).map((x: any) => <A key={x.id} to={'/history/' + x.id}>{day(x.ts)} {x.summary}</A>)}</div></Card>
        <div className="row"><A to={'/history/' + (e.id - 1)} className="btn">◀ Previous</A><span className="right" /><A to={'/history/' + (e.id + 1)} className="btn">Next ▶</A></div>
      </div>
    </div>
  </Shell>;
}

// v13 · Settings (FR-05, FR-06, FR-15–FR-20, FR-63, FR-67, FR-72, FR-90)
// v13 herdr card + v15 setup & doctor (FR-07, FR-08, ADR-009).
function HerdrCard() {
  const { data: st } = useData<any>(() => api.herdr.status(), [], (e) => e.type.startsWith('herdr'));
  return <Card title={<span id="herdr">herdr</span>} right={<A to="/settings/herdr" className="small">Run doctor ›</A>}>
    <div className="row small" style={{ flexWrap: 'wrap', gap: 14 }}>
      {st?.running ? <><span><b>herdr {st.version}</b></span><span>Session <span className="mono">{st.session}</span> <Chip c="grn">running · supervised</Chip></span>
        <span>Socket <span className="mono">{st.socket}</span></span><span>Open it: <code>herdr --session {st.session}</code></span></>
        : <span><Chip c="red">not running</Chip> {st?.error || 'go-build-it starts its herdr session automatically'} · <A to="/settings/herdr">Fix ›</A></span>}
    </div>
  </Card>;
}

export function HerdrSetup() {
  const { data, error, reload } = useData<any>(() => api.herdr.doctor(), [], (e) => e.type.startsWith('herdr'));
  const { run, busy } = useAction();
  const fixes = (data?.checks || []).filter((c: any) => !c.ok || c.warn).length;
  return <Shell title="herdr setup" crumb={<A to="/settings">Settings</A>} right={data && (fixes ? <Chip c="amb">{fixes} to fix</Chip> : <Chip c="grn">all good</Chip>)}>
    <Err msg={error} />
    <Card title="Checks" right={<button type="button" className="lnk small" onClick={reload}>Run again</button>}>
      {!data ? <Loading /> : <table className="t"><tbody>{data.checks.map((c: any) => <tr key={c.name}>
        <td style={{ width: 24 }}>{!c.ok ? '❌' : c.warn ? '⚠️' : '✅'}</td><td><b>{c.name}</b></td><td className="small">{c.detail}</td><td className="small muted">{c.hint}</td>
        <td>{c.fix && <button type="button" className="btn p sm" disabled={busy} onClick={() => run(() => api.herdr.fix({ action: c.fix }), 'Done').then(reload)}>{c.fix.startsWith('install-integration') ? 'Update' : 'Fix'}</button>}</td></tr>)}</tbody></table>}
      {data && <div className="small muted" style={{ marginTop: 8 }}>Missing herdr? <code>{data.installHint}</code>. go-build-it uses its own herdr session; your default session is never touched. CLI: <code>gbi herdr doctor</code></div>}
    </Card>
    <Card title="See it in a terminal"><div className="small">Run <code>herdr --session {data?.status?.session || 'gbi'}</code>: one workspace per project, each member's worktree grouped under it, labelled with role, current task and context. Detach with <kbd>ctrl</kbd>+<kbd>b</kbd> <kbd>q</kbd>; the agents keep running.</div></Card>
  </Shell>;
}

export function Settings() {
  const { data: cfg, reload } = useData<any>(() => api.settings.get(), [], (e) => e.type.startsWith('settings'));
  const { data: harn } = useData<any[]>(() => api.harnesses.list(), []);
  const { data: ts, reload: rts } = useData<any>(() => api.tailscale.status(), []);
  const { data: sys } = useData<any>(() => api.system.info(), []);
  const { run, busy } = useAction();
  if (!cfg) return <Shell title="Settings"><Loading /></Shell>;
  const patch = (p: any, ok = 'Saved') => run(() => api.settings.update(p), ok).then(reload);
  const exp = cfg.experimental || {};
  return <Shell title="Settings" right={<span className="muted small mono">{sys?.dataDir}/config.toml</span>}>
    <HerdrCard />
    <div className="row">
      <Card style={{ width: 180, padding: 10, alignSelf: 'flex-start' }}><div className="col small" style={{ gap: 7 }}>
        {[['herdr', 'herdr'], ['harnesses', 'Agent harnesses'], ['access', 'Access & identity'], ['forges', 'Forge connections'], ['dispatch', 'Dispatch'], ['experimental', 'Experimental'], ['updates', 'Updates']].map(([k, l]) => <a key={k} href={'#' + k}>{l}</a>)}
        <span className="muted" style={{ borderTop: '1px solid var(--line)', paddingTop: 6 }}>Git workflow → <A to="/standards/git-workflow">Standards</A></span></div></Card>
      <div className="col grow">
        <Card title={<span id="harnesses">Agent harnesses</span>} right={<span className="small muted">detected on this machine</span>}>
          <table className="t"><tbody><tr><th>Harness</th><th>Binary</th><th>herdr kind</th><th>Context usage</th><th>Verified in herdr</th><th>Enabled</th></tr>
            {(harn || []).map((h) => <tr key={h.name}><td><b>{h.name}</b><div className="small muted">{h.version}</div></td><td className="mono">{h.path || <span className="muted">{h.binary} (not found)</span>}</td>
              <td>{h.installed ? <Chip c="grn">✓ {h.kind}</Chip> : '—'}</td><td><Chip c={h.usageReported === 'measured' ? 'grn' : 'amb'}>{h.usageReported === 'measured' ? '✓ measured (transcript)' : 'unavailable'}</Chip></td>
              <td>{h.verified ? <Chip c="grn">ADR-009</Chip> : <Chip c="amb">unverified</Chip>}</td>
              <td><Toggle on={h.enabled} label={h.name} onChange={(v) => patch({ harnesses: { ...cfg.harnesses, [h.name]: { ...cfg.harnesses[h.name], enabled: v } } })} /></td></tr>)}</tbody></table>
          <div className="small" style={{ marginTop: 8 }}>Every member runs in herdr. A member keeps its conversation until context reaches the threshold, then rotates (Q8) · compactions are logged · rate limits: <b>pause & auto-resume</b></div>
        </Card>
        <div className="grid" style={{ gridTemplateColumns: '1fr 1fr' }}>
          <Card title={<span id="access">Access & identity</span>}><div className="col small" style={{ gap: 8 }}>
            <div><Toggle on label="local" onChange={() => {}} /> <b>Local</b> {cfg.listen} · no challenge · user <b>{cfg.defaultUser}</b></div>
            <div><Toggle on={!!ts?.serving} label="tailscale" onChange={() => run(() => api.tailscale.setup(), 'Tailscale Serve configured').then(rts)} /> <b>Tailscale Serve</b> {ts?.loggedIn ? <Chip c="grn">logged in</Chip> : <Chip c="amb">{ts?.installed ? 'not logged in' : 'not installed'}</Chip>}
              {ts?.serveUrl && <div className="mono"><a href={ts.serveUrl} target="_blank" rel="noreferrer">{ts.serveUrl}</a> · identity from Tailscale headers</div>}
              {ts?.guidance && <div className="muted">{ts.guidance}</div>}{ts?.loggedIn && !ts?.serving && <button type="button" className="btn sm" disabled={busy} onClick={() => run(() => api.tailscale.setup(), 'Serving on your tailnet').then(rts)}>Set up Tailscale Serve</button>}</div>
            <div><Toggle on={false} label="external" onChange={() => {}} /> <b>External interfaces</b> off · <span className="muted">refused in the MVP (ADR-005)</span></div>
            <div className="muted">You are: {sys?.actor?.name} via {sys?.actor?.origin}</div></div></Card>
          <Card title={<span id="forges">Forge connections</span>}><div className="col small" style={{ gap: 8 }}>
            <div>GitHub (<code>gh</code>) and GitLab (<code>glab</code>) are detected per project from its remote; Courier checks they are authenticated before pushing (FR-90).</div>
            <div className="muted">The workflow (branching, PR rules, merge strategy) is a global standard. <A to="/standards/git-workflow">Edit in Standards → Git workflow ›</A></div></div></Card>
          <Card title={<span id="dispatch">Dispatch</span>}><div className="col small" style={{ gap: 8 }}>
            <div><Toggle on={!!cfg.autoDispatch} label="auto dispatch" onChange={(v) => run(() => api.dispatcher.set({ auto: v }), v ? 'Automatic dispatch on' : 'Automatic dispatch off').then(reload)} /> <b>Automatic dispatch</b>: start ready tasks without asking</div>
            <div className="row c">Max concurrent sessions <input type="number" min={1} max={10} defaultValue={cfg.maxConcurrent} style={{ width: 60 }} onBlur={(e) => patch({ maxConcurrent: +e.target.value })} /></div></div></Card>
          <Card title={<span id="experimental">Experimental <Chip c="blue">off by default</Chip></span>} style={{ borderColor: 'var(--blue)' }}><div className="col small" style={{ gap: 10 }}>
            <div><Toggle on={!!exp.smartHandoffs} label="smart handoffs" onChange={(v) => patch({ experimental: { ...exp, smartHandoffs: v } })} /> <b>Smart handoffs</b> (FR-72–74) · threshold <input type="number" min={50} max={95} defaultValue={Math.round((exp.handoffThreshold || .75) * 100)} style={{ width: 54 }} onBlur={(e) => patch({ experimental: { ...exp, handoffThreshold: +e.target.value / 100 } })} />%
              <div className="muted">Ask for a curated briefing at the threshold instead of compacting. Projects (blank = all): <input defaultValue={(exp.smartHandoffProjects || []).join(', ')} onBlur={(e) => patch({ experimental: { ...exp, smartHandoffProjects: e.target.value.split(',').map((s) => s.trim()).filter(Boolean) } })} style={{ width: 160 }} /></div></div></div></Card>
          <Card title={<span id="updates">Updates</span>}><div className="col small" style={{ gap: 6 }}>
            <div>Current <b>{sys?.version}</b> · channel {cfg.updates?.channel} · <Toggle on={!!cfg.updates?.autoInstall} label="auto install" onChange={(v) => patch({ updates: { ...cfg.updates, autoInstall: v } })} /> auto-install</div>
            <div className="muted">Automatic updates wait until no member is working; a manual update can run any time, because herdr keeps members running while go-build-it restarts. Checksum and Ed25519 signature are verified (ADR-004, ADR-011).</div>
            <div>{sys?.update?.updating ? <b>Updating…</b> : sys?.update?.newer ? <>Update ready: <b>{sys.update.latest}</b></> : sys?.update?.latest ? 'Up to date' : null}</div>
            <div><A to="/changelog">Change log &amp; update now ›</A> <span className="muted">(<kbd>g</kbd> <kbd>v</kbd>)</span></div></div></Card>
        </div>
      </div>
    </div>
  </Shell>;
}

// v14 · Embedded documentation (FR-130–FR-133)
export function Docs({ slug }: { slug?: string }) {
  const { data: pages } = useData<any[]>(() => api.docs.list(), []);
  const [q, setQ] = useState('');
  const [hits, setHits] = useState<any[]>([]);
  const cur = slug || pages?.[0]?.slug;
  const { data: page } = useData<any>(() => (cur ? api.docs.get(cur) : Promise.resolve(null)), [cur]);
  useEffect(() => { const t = setTimeout(() => { if (q) api.docs.search({ q }).then(setHits); else setHits([]); }, 150); return () => clearTimeout(t); }, [q]);
  const sections: Record<string, any[]> = {};
  (pages || []).forEach((p) => { (sections[p.section || 'Other'] ||= []).push(p); });
  const body = (page?.body || '').replace(/\]\(screenshots\//g, '](/docs-assets/screenshots/');
  return <Shell title="Docs" right={<input placeholder="Search docs…" value={q} onChange={(e) => setQ(e.target.value)} style={{ width: 220 }} />}>
    <div className="row">
      <Card title="Contents" style={{ width: 230, padding: 10 }}>{Object.entries(sections).map(([s, ps]) => <div key={s} style={{ marginBottom: 8 }}><div className="muted xs" style={{ textTransform: 'uppercase' }}>{s}</div>
        {ps.map((p) => <div key={p.slug} className="small" style={{ padding: '2px 0' }}><A to={'/docs/' + p.slug}>{p.slug === cur ? <b>▸ {p.title}</b> : p.title}</A></div>)}</div>)}</Card>
      <Card className="grow">{page ? <>{page.reqs && <div className="small muted" style={{ marginBottom: 8 }}>Requirements {page.reqs} · screenshots captured by the UAT run</div>}<Markdown src={body} /></> : <Loading />}</Card>
      {hits.length > 0 && <Card title={<>Search results · "{q}"</>} style={{ width: 260 }}><div className="col small" style={{ gap: 8 }}>{hits.map((h) => <A key={h.page.slug} to={'/docs/' + h.page.slug}><b>{h.page.title}</b><div className="muted">{h.snippet}</div></A>)}</div></Card>}
    </div>
  </Shell>;
}

export { ago };
