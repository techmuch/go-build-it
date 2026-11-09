import { useEffect, useMemo, useState } from 'react';
import { api } from '../api.gen';
import { useData } from '../live';
import { navigate, query } from '../router';
import { ProjectTabs, Shell } from '../shell';
import { A, AgentChat, Card, Chip, DictArea, Editor, Empty, Err, Loading, Markdown, Proposals, Viewer, applyHunks, day, kindOf, useAction, type Hunk } from '../ui';

import { usePersonas } from './Team';

// v03 · Project artifacts, problem definition & values (FR-45, FR-46, FR-123)
export function Artifacts({ id }: { id: string }) {
  const { data: list, error } = useData<any[]>(() => api.artifacts.list(id), [id], (e) => e.type.startsWith('artifact'));
  const sel = query('path') || (list || []).find((a) => a.path.endsWith('requirements.md'))?.path || list?.[0]?.path || '';
  const { data: art } = useData<any>(() => (sel ? api.artifacts.get(id, sel) : Promise.resolve(null)), [id, sel], (e) => e.subject === sel);
  const { data: vals } = useData<any[]>(() => api.values.list(id), [id], (e) => e.type.startsWith('value'));
  const { data: prob } = useData<any>(() => api.problem.get(id), [id], (e) => e.type.startsWith('problem'));
  const [viewer, setViewer] = useState<number | null>(null);
  const [rev, setRev] = useState('');
  const [revText, setRevText] = useState<string | null>(null);
  const [newVal, setNewVal] = useState('');
  const [probEdit, setProbEdit] = useState<string | null>(null);
  const { run } = useAction();
  const groups = useMemo(() => { const g: Record<string, any[]> = {}; (list || []).forEach((a) => { (g[a.group] ||= []).push(a); }); return g; }, [list]);
  const visuals = (list || []).filter((a) => a.kind === 'html' || a.kind === 'image');
  useEffect(() => { setRev(''); setRevText(null); }, [sel]);
  const showRev = async (h: string) => { setRev(h); const r = await api.artifacts.get(id, sel, { rev: h }); setRevText(r.content); };
  return <Shell title="Artifacts" crumb={<A to={`/projects/${id}`}>{id}</A>} tabs={<ProjectTabs id={id} on="artifacts" />}>
    <Err msg={error} />
    <div className="row">
      <Card title="Documents" style={{ width: 230, padding: 10 }}>
        {Object.entries(groups).map(([g, items]) => <div key={g} style={{ marginBottom: 10 }}>
          <div className="muted xs" style={{ textTransform: 'uppercase', margin: '4px 0' }}>{g}</div>
          {items.map((a) => <div key={a.path} className="small" style={{ padding: '2px 0', overflowWrap: 'anywhere' }}>
            {a.kind === 'html' || a.kind === 'image' ? <button type="button" className="lnk" onClick={() => setViewer(visuals.findIndex((v) => v.path === a.path))}>{a.path.split('/').pop()} ⤢</button>
              : <A to={`/projects/${id}/artifacts?path=${encodeURIComponent(a.path)}`}>{a.path === sel ? <b>▸ {a.path.split('/').pop()}</b> : a.path.split('/').pop()}</A>}</div>)}
        </div>)}
        {!list?.length && <span className="muted small">No docs/ folder yet. Create one with the editor.</span>}
        <A to={`/projects/${id}/edit?path=docs/new.md`} className="btn" >+ New document</A>
      </Card>
      <Card className="grow" title={<span className="mono" style={{ textTransform: 'none' }}>{sel || 'No document'}</span>}
        right={sel && art && <span className="small">Rev {art.rev} {art.revisions?.[0] && <>· {art.revisions[0].author}</>} · <A to={`/projects/${id}/edit?path=${encodeURIComponent(sel)}`} className="btn p sm">✎ Edit</A></span>}>
        {!art ? <Loading /> : <>
          {rev && <div className="note" style={{ marginBottom: 8 }}>Showing revision {rev}. <button type="button" className="lnk" onClick={() => { setRev(''); setRevText(null); }}>Back to current</button></div>}
          {art.kind === 'markdown' ? <Markdown src={revText ?? art.content} /> : <pre className="mono" style={{ whiteSpace: 'pre-wrap' }}>{revText ?? art.content}</pre>}
          <details style={{ marginTop: 14 }}><summary className="small muted">Revision history ({art.revisions?.length || 0})</summary>
            <table className="t small"><tbody>{(art.revisions || []).map((r: any, k: number) => <tr key={r.hash}><td className="mono">{r.hash}</td><td>Rev {art.revisions.length - k}</td><td>{r.author}</td><td>{r.date}</td><td>{r.subject}</td><td><button type="button" className="lnk" onClick={() => showRev(r.hash)}>view</button></td></tr>)}</tbody></table></details>
        </>}
      </Card>
      <div className="col" style={{ width: 300 }}>
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
      </div>
    </div>
    {viewer !== null && viewer >= 0 && <Viewer items={visuals.map((v) => ({ name: v.path, kind: v.kind, url: `/raw/${id}/${v.path}` }))} index={viewer} onClose={() => setViewer(null)} title={id} projectId={id} />}
  </Shell>;
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
  const save = () => run(() => api.artifacts.save(id, path, { content: text, message: msg || undefined }), `Saved as Rev ${rev + 1}`).then((r) => { if (r) { setBase(text); setRev(r.rev); navigate(`/projects/${id}/artifacts?path=${encodeURIComponent(path)}`); } });
  return <Shell title={<>Editing {path.split('/').pop()}</>} crumb={<><A to={`/projects/${id}`}>{id}</A> / <A to={`/projects/${id}/artifacts?path=${encodeURIComponent(path)}`}>Artifacts</A></>}
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
          <A to={`/projects/${id}/artifacts?path=${encodeURIComponent(path)}`} className="btn">Discard</A>
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
