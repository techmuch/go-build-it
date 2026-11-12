import { useCallback, useEffect, useState } from 'react';
import { api } from '../api.gen';
import { useData } from '../live';
import { navigate, query } from '../router';
import { Shell } from '../shell';
import { A, AgentChat, Av, Card, Chip, Editor, Empty, Err, Loading, Markdown, Proposals, Status, Toggle, ago, applyHunks, day, useAction, useToast, type Hunk } from '../ui';
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

type DoctorResult = { id: string; tier: string; status: 'ok' | 'warn' | 'fail' | 'skipped'; title: string; detail: string; harness?: string; evidence?: { command?: string; exit?: number; output?: string }; fix?: { kind: 'auto' | 'human'; action?: string; instructions: string }; docs: string; ms: number };
type DoctorReport = { results: DoctorResult[]; tiers: { tier: string; ran: boolean; ok: number; total: number; note?: string }[]; exit: number; summary: string; ms: number; started: string };
// Inline renders `code` spans in a fix's instructions.
function Inline({ text }: { text: string }) {
  return <>{text.split(/(`[^`]+`)/).map((p, i) => p.startsWith('`') && p.endsWith('`') ? <code key={i}>{p.slice(1, -1)}</code> : <span key={i}>{p}</span>)}</>;
}
const DICON: Record<DoctorResult['status'], string> = { ok: '✅', warn: '⚠️', fail: '✖', skipped: '–' };

// v15 + v17 · herdr setup and the doctor (FR-07a–FR-07e, ADR-012): tiers, evidence, automatic and human fixes.
export function HerdrSetup() {
  const [opts, setOpts] = useState<{ through?: string; prompt?: boolean }>({});
  const [data, setData] = useState<{ report: DoctorReport; status: any; installHint: string } | undefined>(undefined);
  const [error, setError] = useState('');
  const [running, setRunning] = useState(false);
  const { run, busy } = useAction();
  const toast = useToast();
  const go = useCallback(async (o: { through?: string; prompt?: boolean } = {}) => {
    setRunning(true); setError(''); setOpts(o);
    try { setData(await api.doctor.run(o as any)); } catch (e: any) { setError(String(e.message || e)); } finally { setRunning(false); }
  }, []);
  useEffect(() => { go(); }, [go]);
  const rep = data?.report;
  const fails = rep ? rep.results.filter((r) => r.status === 'fail').length : 0;
  const human = rep ? rep.results.filter((r) => r.status === 'fail' && r.fix?.kind !== 'auto').length : 0;
  const copyReport = async () => {
    const r: any = await run(() => api.doctor.report(opts as any));
    if (!r) return;
    try { await navigator.clipboard.writeText(r.markdown); toast('Report copied: paste it into an AI chat or an issue'); } catch { setData((d) => d && { ...d, md: r.markdown } as any); }
  };
  const md: string | undefined = (data as any)?.md;
  return <Shell title="herdr setup" crumb={<A to="/settings">Settings</A>} right={rep && (human ? <Chip c="red">{human} needs you</Chip> : fails ? <Chip c="amb">{fails} to fix</Chip> : <Chip c="grn">all good</Chip>)}>
    <Err msg={error} />
    <Card title="Doctor" right={<span className="muted small">{running ? 'running…' : rep ? `ran ${ago(rep.started)} · ${(rep.ms / 1000).toFixed(1)}s` : ''}</span>}>
      <div className="tiers" data-testid="doctor-tiers">
        {(rep?.tiers || []).map((t) => <span key={t.tier} className={'tierchip ' + (!t.ran ? 'off' : t.ok === t.total ? 'ok' : 'bad')} data-tier={t.tier} title={t.note}>
          {!t.ran ? '' : t.ok === t.total ? '✅ ' : '✖ '}{t.tier} {t.ran ? `${t.ok}/${t.total}` : `· ${t.tier === 'prompt' ? 'not run' : (t.note || '').replace(/^skipped: /, 'skipped')}`}</span>)}
        <span className="grow" />
        <button type="button" className="btn sm" disabled={running} onClick={() => go()}>Run again</button>
        <button type="button" className="btn sm" disabled={running} onClick={() => go({ through: 'harness' })} data-testid="doctor-harness">Test harnesses</button>
        <button type="button" className="btn sm" disabled={running} onClick={() => go({ through: 'harness', prompt: true })} title="Sends one short prompt to each harness" data-testid="doctor-prompt">Send a test prompt · uses tokens</button>
        <button type="button" className="btn sm" disabled={running || busy} onClick={copyReport}>Copy report</button>
      </div>
      {running && !rep && <Loading what="the doctor" />}
      {rep && <table className="t doctor" aria-busy={running}><tbody>{rep.results.map((r, i) => <tr key={r.id + (r.harness || '') + i} className={'d-' + r.status} data-check={r.id} data-status={r.status}>
        <td style={{ width: 24 }} aria-label={r.status}>{DICON[r.status]}</td>
        <td><b>{r.title}</b>{r.status !== 'ok' && <div className="mono small muted">{r.id}</div>}</td>
        <td className="small">{r.detail}
          {(r.status === 'fail' || r.status === 'warn') && r.evidence && (r.evidence.command || r.evidence.output) && <pre className="evi">{r.evidence.command ? '$ ' + r.evidence.command + '\n' : ''}{r.evidence.exit !== undefined && r.status === 'fail' ? `exit ${r.evidence.exit}\n` : ''}{r.status === 'fail' ? r.evidence.output || '' : ''}</pre>}
          {r.fix && r.status !== 'ok' && (r.fix.kind === 'human'
            ? <div className="hfix" data-testid="human-fix"><b>Needs you:</b> <Inline text={r.fix.instructions} /> <A to={r.docs}>Troubleshooting: {r.id}</A></div>
            : <div className="small muted" style={{ marginTop: 4 }}>{r.fix.instructions} <A to={r.docs}>More</A></div>)}
        </td>
        <td className="small muted" style={{ whiteSpace: 'nowrap' }}>{r.ms ? (r.ms / 1000).toFixed(1) + 's' : ''}</td>
        <td>{r.fix?.kind === 'auto' && r.status !== 'ok' && <button type="button" className="btn p sm" disabled={busy || running} onClick={() => run(() => api.doctor.fix({ action: r.fix!.action }), 'Fixed').then((d: any) => d && setData(d))}>{r.fix.action?.startsWith('install-integration') ? 'Update' : 'Fix'}</button>}</td>
      </tr>)}</tbody></table>}
      {rep && <div className="small muted" style={{ marginTop: 8 }} data-testid="doctor-summary">{rep.summary}. The doctor's throwaway panes live in the <span className="mono">gbi-doctor</span> workspace and are closed afterwards; your default herdr session is never touched. Same checks without the server: <code>gbi doctor</code> · for an AI agent: <code>gbi doctor --json --fix</code>.</div>}
      {md && <pre className="evi" style={{ maxHeight: 240, overflow: 'auto' }}>{md}</pre>}
    </Card>
    <Card title="See it in a terminal"><div className="small">Run <code>herdr --session {data?.status?.session || 'gbi'}</code>: one workspace per project, each member's worktree grouped under it, labelled with role, current task and context. Detach with <kbd>ctrl</kbd>+<kbd>b</kbd> <kbd>q</kbd>; the agents keep running. Missing herdr? <code>{data?.installHint}</code></div></Card>
  </Shell>;
}

// Remote access (FR-20, FR-20a–FR-20c): is go-build-it served on the tailnet, is it this
// go-build-it, is it private, what else this machine serves, and a QR code for a phone.
const CHECK_ICON: Record<string, string> = { ok: '✅', warn: '⚠️', fail: '✖', skip: '–' };

export function QRCode({ rows, size = 168, label }: { rows: string[]; size?: number; label: string }) {
  const n = rows.length;
  const d = rows.map((r, y) => [...r].map((c, x) => (c === '1' ? `M${x} ${y}h1v1h-1z` : '')).join('')).join('');
  return <svg role="img" aria-label={label} viewBox={`0 0 ${n} ${n}`} width={size} height={size} shapeRendering="crispEdges" style={{ background: '#fff', borderRadius: 6 }} data-testid="remote-qr">
    <rect width={n} height={n} fill="#fff" /><path fill="#000" d={d} /></svg>;
}

function RemoteAccessCard({ isOwner }: { isOwner: boolean }) {
  const { data: ts, reload } = useData<any>(() => api.tailscale.status(), []);
  const { run, busy } = useAction();
  const p = ts?.proposal;
  const others = (ts?.rules || []).filter((r: any) => !r.mine);
  const owner = isOwner ? '' : 'Only the owner can change how go-build-it is served';
  return <Card title={<span id="remote">Remote access</span>} right={<span className="row c" style={{ gap: 6 }}>
      {ts && <Chip c={ts.reachable && ts.private ? 'grn' : ts.serving ? 'red' : 'amb'}>{ts.reachable && ts.private ? 'working' : ts.serving ? 'needs attention' : 'not served'}</Chip>}
      <button type="button" className="btn sm" disabled={busy} onClick={() => reload()}>Check again</button></span>}>
    {!ts ? <Loading /> : <div className="row" style={{ gap: 18, alignItems: 'flex-start', flexWrap: 'wrap' }}>
      <div className="col grow" style={{ gap: 10, minWidth: 280 }}>
        <table className="t small" data-testid="remote-checks"><tbody>
          {(ts.checks || []).map((c: any) => <tr key={c.id} data-check={c.id} data-status={c.status}>
            <td style={{ width: 22 }}>{CHECK_ICON[c.status] || '·'}</td><td><b>{c.title}</b>{c.detail && <div className="muted">{c.detail}</div>}</td>
            <td>{c.fix && c.status !== 'ok' && <code className="small">{c.fix}</code>}</td></tr>)}</tbody></table>
        {ts.serving && <div className="row c small" style={{ gap: 8 }}>
          <span className="mono"><a href={ts.serveUrl} target="_blank" rel="noreferrer">{ts.serveUrl}</a></span><span className="muted">· tailnet only · identity from Tailscale</span>
          <span style={{ marginLeft: 'auto' }} /><button type="button" className="btn sm" title={owner} disabled={busy || !isOwner} onClick={() => run(() => api.tailscale.stop(), 'Stopped serving go-build-it').then(reload)}>Stop serving</button></div>}
        {ts.loggedIn && !ts.serving && p && <div className="col small" style={{ gap: 6 }} data-testid="remote-proposal">
          {p.needsConfirm && <div><Chip c="amb">address taken</Chip> {p.reason}</div>}
          <div>go-build-it will run <code>{p.command}</code> and be at <b className="mono">{p.url}</b>.</div>
          <div><button type="button" className="btn p sm" title={owner} disabled={busy || !isOwner} onClick={() => run(() => api.tailscale.setup({ port: p.port, confirm: true }), 'Serving on your tailnet').then(reload)}>{p.needsConfirm ? `Serve on its own port ${p.port}` : 'Serve on my tailnet'}</button>
            <span className="muted"> · never replaces another service's address</span></div></div>}
        {!ts.loggedIn && ts.guidance && <div className="small muted">{ts.guidance}</div>}
        {others.length > 0 && <details className="small" open={(ts.conflicts || []).length > 0}><summary>Also served from this machine ({others.length})</summary>
          <table className="t small" data-testid="remote-others"><tbody><tr><th>Tailnet address</th><th>Goes to</th><th>Program</th><th /></tr>
            {others.map((r: any) => <tr key={r.url}><td className="mono">{r.url}</td><td className="mono">{r.target}</td><td>{r.process || <span className="muted">unknown</span>}</td>
              <td>{r.funnel ? <Chip c="red">public (Funnel)</Chip> : <Chip>tailnet</Chip>}</td></tr>)}</tbody></table></details>}
      </div>
      {ts.qr?.length > 0 && <div className="col" style={{ gap: 6, alignItems: 'center', width: 190 }}>
        <QRCode rows={ts.qr} label={'QR code for ' + (ts.phoneUrl || ts.serveUrl)} />
        <div className="small muted" style={{ textAlign: 'center' }}>Opens the mobile UI, <a href={ts.phoneUrl} className="mono" style={{ overflowWrap: 'anywhere' }}>{ts.phoneUrl}</a>. Scan with a phone signed in to the same tailnet. Terminal: <code>gbi tailscale qr</code></div></div>}
    </div>}
  </Card>;
}

export function Settings() {
  const { data: cfg, reload } = useData<any>(() => api.settings.get(), [], (e) => e.type.startsWith('settings'));
  const { data: harn } = useData<any[]>(() => api.harnesses.list(), []);
  const { data: sys } = useData<any>(() => api.system.info(), []);
  const { run } = useAction();
  if (!cfg) return <Shell title="Settings"><Loading /></Shell>;
  const patch = (p: any, ok = 'Saved') => run(() => api.settings.update(p), ok).then(reload);
  const exp = cfg.experimental || {};
  return <Shell title="Settings" right={<span className="muted small mono">{sys?.dataDir}/config.toml</span>}>
    <HerdrCard />
    <div className="row">
      <Card style={{ width: 180, padding: 10, alignSelf: 'flex-start' }}><div className="col small" style={{ gap: 7 }}>
        {[['herdr', 'herdr'], ['harnesses', 'Agent harnesses'], ['remote', 'Remote access'], ['access', 'Access & identity'], ['forges', 'Forge connections'], ['dispatch', 'Dispatch'], ['experimental', 'Experimental'], ['updates', 'Updates']].map(([k, l]) => <a key={k} href={'#' + k}>{l}</a>)}
        <span className="muted" style={{ borderTop: '1px solid var(--line)', paddingTop: 6 }}>Git workflow → <A to="/standards/git-workflow">Standards</A></span></div></Card>
      <div className="col grow">
        <Card title={<span id="harnesses">Agent harnesses</span>} right={<span className="small muted">detected on this machine</span>}>
          <table className="t"><tbody><tr><th>Harness</th><th>Binary</th><th>herdr kind</th><th>Context usage</th><th>Verified in herdr</th><th>Enabled</th></tr>
            {(harn || []).map((h) => <tr key={h.name}><td><b>{h.name}</b><div className="small muted">{h.version}</div></td><td className="mono">{h.path || <span className="muted">{h.binary} (not found)</span>}</td>
              <td>{h.installed ? <Chip c="grn">✓ {h.kind}</Chip> : '—'}</td><td><Chip c={h.usageReported === 'measured' ? 'grn' : 'amb'}>{h.usageReported === 'measured' ? '✓ measured (transcript)' : 'unavailable'}</Chip></td>
              <td>{h.verified ? <Chip c="grn">ADR-009</Chip> : <Chip c="amb">unverified</Chip>}</td>
              <td><Toggle on={h.enabled} label={h.name} onChange={(v) => patch({ harnesses: { ...cfg.harnesses, [h.name]: { ...cfg.harnesses[h.name], enabled: v } } })} /></td></tr>)}</tbody></table>
          <div className="small" style={{ marginTop: 8 }}><Toggle on={!!cfg.agents?.trustFolders} label="answer folder trust prompts" onChange={(v) => patch({ agents: { ...cfg.agents, trustFolders: v } }, v ? 'go-build-it will answer folder trust prompts' : 'Folder trust prompts will wait for you')} /> <b>Answer folder trust prompts</b> · say yes when a harness asks to trust a folder go-build-it created (member worktrees, helpers, the practice project). Sign-in and permission prompts still come to you (FR-07f).</div>
          <div className="small" style={{ marginTop: 8 }}>Every member runs in herdr. A member keeps its conversation until context reaches the threshold, then rotates (Q8) · compactions are logged · rate limits: <b>pause & auto-resume</b></div>
        </Card>
        <RemoteAccessCard isOwner={!!sys?.isOwner} />
        <div className="grid" style={{ gridTemplateColumns: '1fr 1fr' }}>
          <Card title={<span id="access">Access & identity</span>}><div className="col small" style={{ gap: 8 }}>
            <div><Toggle on label="local" onChange={() => {}} /> <b>Local</b> {cfg.listen} · no challenge · user <b>{cfg.defaultUser}</b></div>
            <div><b>Tailscale</b> · people on your tailnet are identified by Tailscale's headers. <a href="#remote">Remote access ›</a></div>
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
  useEffect(() => { // links such as /docs/troubleshooting#harness.starts (FR-07e)
    const id = decodeURIComponent(window.location.hash.slice(1));
    if (page && id) window.setTimeout(() => document.getElementById(id)?.scrollIntoView({ block: 'start' }), 50);
  }, [page]);
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
