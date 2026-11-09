import { useEffect, useMemo, useState } from 'react';
import { api } from '../api.gen';
import { useData } from '../live';
import { navigate } from '../router';
import { ProjectTabs, Shell } from '../shell';
import { A, AgentChat, Av, Card, Chip, DiffView, Editor, Empty, Err, Loading, Proposals, Seg, applyHunks, useAction, type Hunk } from '../ui';
import { clickable } from '../keys';

/** Personas an agent chat can use: the project's AI members, or the archetypes for global work. */
export function usePersonas(projectId?: string) {
  const { data } = useData<any[]>(() => (projectId ? api.members.list(projectId) : api.archetypes.list()), [projectId], (e) => /^(member|archetype)\./.test(e.type));
  return (data || []).filter((m) => (projectId ? m.kind === 'ai' : m.harness !== 'courier')).map((m) => ({ id: m.id, name: m.name, role: m.role }));
}

// Parse persona Markdown back into instructions + role standards (FR-13).
export function parsePersona(md: string) {
  const body: string[] = []; const stds: string[] = [];
  let inStd = false;
  for (const l of md.split('\n')) {
    if (/^# /.test(l)) continue;
    if (/^## Role standards/i.test(l)) { inStd = true; continue; }
    if (inStd && /^- /.test(l)) { stds.push(l.slice(2).trim()); continue; }
    if (inStd && /^## /.test(l)) inStd = false;
    if (!inStd) body.push(l);
  }
  return { instructions: body.join('\n').trim(), standards: stds };
}

function lineDiff(a: string, b: string): string {
  // Small unified diff for display: removed lines from a, added lines from b.
  const A = a.split('\n'), B = b.split('\n');
  const setA = new Set(A), setB = new Set(B);
  const out = ['@@ archetype → effective @@'];
  for (const l of A) if (!setB.has(l) && l.trim()) out.push('-' + l);
  for (const l of B) out.push((setA.has(l) ? ' ' : '+') + l);
  return out.join('\n');
}

const harnessOptions = (harn: any[] | undefined) => (harn || []).map((h) => h.name).concat(['fake']);

// v10 · Persona archetypes (global) with change review and coach (FR-12, FR-12b, FR-12c, FR-13, FR-25, FR-26)
export function Archetypes({ aid }: { aid?: string }) {
  const { data: list } = useData<any[]>(() => api.archetypes.list(), [], (e) => /^(archetype|member)\./.test(e.type));
  const { data: roles } = useData<any[]>(() => api.roles.list(), [], (e) => e.type.startsWith('role'));
  const { run } = useAction();
  const sel = aid || list?.[0]?.id || '';
  return <Shell title="Persona archetypes" right={<span className="small muted">Templates for project team members · <A to="/docs/05-team">How teams work</A></span>}>
    <div className="row">
      <Card style={{ width: 260, padding: '6px 10px' }}>
        <table className="t small"><tbody><tr><th /><th>Archetype</th><th>Used by</th></tr>
          {(list || []).map((a) => <tr key={a.id} className={'click ' + (a.id === sel ? 'sel' : '')} {...clickable(() => navigate('/archetypes/' + a.id), { role: 'link' })}>
            <td><Av name={a.name} role={a.role} /></td><td><b>{a.name}</b> <span className="muted">· {(roles || []).find((r) => r.id === a.role)?.name || a.role}</span><br /><span className="muted">{a.harness === 'courier' ? 'built-in Courier' : `${a.harness}${a.model ? ' · ' + a.model : ''}`} · rev {a.rev}</span></td><td>{a.usedBy}</td></tr>)}</tbody></table>
        {!list && <Loading />}
        <div className="row c wrap" style={{ gap: 6, margin: '8px 0' }}><A to="/archetypes/new" className="btn">+ Archetype</A>
          <button type="button" className="btn" onClick={() => { const n = window.prompt('Custom role name'); if (n) run(() => api.roles.add({ id: n.toLowerCase().replace(/\W+/g, '-'), name: n }), 'Role added'); }}>+ Role</button></div>
      </Card>
      {sel && <ArchetypeEditor key={sel} id={sel} roles={roles || []} />}
    </div>
  </Shell>;
}

function ArchetypeEditor({ id, roles }: { id: string; roles: any[] }) {
  const isNew = id === 'new';
  const personas = usePersonas();
  const { data: harn } = useData<any[]>(() => api.harnesses.list(), []);
  const { data, reload } = useData<any>(() => (isNew ? Promise.resolve(null) : api.archetypes.get(id)), [id], (e) => /^(archetype|member)\./.test(e.type));
  const [a, setA] = useState<any>(isNew ? { id: '', name: '', role: 'developer', harness: 'claude', model: '', instructions: '', standards: [] } : null);
  const [doc, setDoc] = useState(isNew ? '# New archetype\n\nDescribe how members made from this archetype work.\n' : '');
  const [mode, setMode] = useState<'wysiwyg' | 'markdown' | 'split'>('markdown');
  const [coach, setCoach] = useState('');
  const [prop, setProp] = useState<{ hunks: Hunk[]; base: string } | null>(null);
  const [acc, setAcc] = useState<Set<number>>(new Set());
  const { run, busy } = useAction();
  useEffect(() => { if (data?.archetype) { setA(data.archetype); setDoc(data.persona); } }, [data?.archetype?.rev, data?.archetype?.id]);
  useEffect(() => { if (!coach && personas.length) setCoach((personas.find((p) => p.role === 'architect') || personas[0]).id); }, [personas, coach]);
  if (!a) return <Loading />;
  const save = () => {
    const p = parsePersona(doc);
    return run(() => api.archetypes.save({ ...a, instructions: p.instructions, standards: p.standards }), `Saved ${a.name}`).then((r: any) => { if (r) { if (isNew) navigate('/archetypes/' + r.archetype.id); else reload(); } });
  };
  const changes: any[] = data?.changes || [];
  const clean = changes.filter((c) => c.canApplyClean);
  const field = (label: string, el: React.ReactNode) => <label className="small" style={{ display: 'flex', gap: 6, alignItems: 'center' }}>{label} {el}</label>;
  return <>
    <div className="col grow">
      <Card style={{ padding: '10px 14px' }}><div className="row c wrap" style={{ gap: 10 }}>
        <Av name={a.name || '?'} role={a.role} />
        <input aria-label="Name" placeholder="Name" value={a.name} onChange={(e) => setA({ ...a, name: e.target.value })} style={{ width: 120, fontWeight: 700 }} />
        {field('Role', <select value={a.role} onChange={(e) => setA({ ...a, role: e.target.value })}>{roles.map((r) => <option key={r.id} value={r.id}>{r.name}</option>)}</select>)}
        {a.harness === 'courier' ? <Chip>built-in Courier</Chip> : <>{field('Default harness', <select value={a.harness} onChange={(e) => setA({ ...a, harness: e.target.value })}>{harnessOptions(harn).map((h) => <option key={h}>{h}</option>)}</select>)}
          {field('Default model', <input value={a.model} onChange={(e) => setA({ ...a, model: e.target.value })} placeholder="harness default" style={{ width: 140 }} />)}</>}
        {!isNew && <span className="small muted">rev {a.rev} · used by {a.usedBy} member{a.usedBy === 1 ? '' : 's'}</span>}
      </div></Card>
      {prop && <><Proposals hunks={prop.hunks} accepted={acc} setAccepted={setAcc} /><div className="row c"><span className="right" /><button type="button" className="btn" onClick={() => setProp(null)}>Discard</button>
        <button type="button" className="btn p" onClick={async () => { setDoc(await applyHunks(prop.base, prop.hunks, acc)); setProp(null); }}>Apply {acc.size} accepted</button></div></>}
      <Editor value={doc} onChange={setDoc} mode={mode} onMode={setMode} toolbarRight={<span className="mono">archetypes/{a.id || 'new'}.md</span>} />
      <span className="small muted">Bullets under “## Role standards” are added on top of the global standards. Saving makes a new revision; members move to it only when you apply it (FR-12b).</span>
      <div className="row c">{!isNew && <button type="button" className="btn d" disabled={a.usedBy > 0} title={a.usedBy ? 'Members are made from this archetype' : ''} onClick={() => { if (window.confirm('Remove ' + a.name + '?')) run(() => api.archetypes.remove(id), 'Removed').then(() => navigate('/archetypes')); }}>Remove</button>}
        <span className="right" /><button type="button" className="btn p" disabled={busy || !a.name} onClick={save}>{isNew ? 'Create archetype' : `Save rev ${(a.rev || 0) + 1}`}</button></div>
    </div>
    <div className="col" style={{ width: 320 }}>
      {!isNew && <Card title={`Change review · rev ${a.rev}`}>
        {changes.length === 0 ? <span className="small muted">Every member made from {a.name} is on rev {a.rev}.</span> : <>
          <div className="small">{changes.length} member{changes.length === 1 ? ' is' : 's are'} behind:</div>
          <table className="t small"><tbody>{changes.map((c) => <tr key={c.member.id}><td><A to={`/projects/${c.member.projectId}/team/${encodeURIComponent(c.member.id)}`}>{c.member.name}</A> <span className="muted">· {c.member.projectId} · rev {c.fromRev}</span></td>
            <td>{c.canApplyClean ? <Chip c="grn">no override</Chip> : <Chip c="amb">overrides {c.overrides.join(', ')}</Chip>}</td>
            <td><button type="button" className="lnk" onClick={() => run(() => api.archetypes.apply(id, { member: c.member.id }), `Applied to ${c.member.name}`).then(reload)}>{c.canApplyClean ? 'Apply' : 'Apply anyway'}</button></td></tr>)}</tbody></table>
          {clean.length > 0 && <button type="button" className="btn p" disabled={busy} onClick={() => run(() => api.archetypes.apply(id, { all: true }), `Applied to ${clean.length} members`).then(reload)}>Apply to {clean.length} member{clean.length === 1 ? '' : 's'} without overrides</button>}
          <div className="small muted">Members keep their project tuning. Overridden parts stay until you review them. Nothing is pushed silently.</div></>}
      </Card>}
      {!isNew && <AgentChat title="Persona coach" personas={personas} persona={coach} setPersona={setCoach} placeholder={`Describe how a ${a.name} should behave…`}
        suggestions={['Interview me', 'Check against standards', 'Learn from history']}
        send={(msg, h) => api.members.chat(id, { coach, message: msg, content: doc, history: h })}
        onProposal={(_d, hunks) => { setProp({ hunks, base: doc }); setAcc(new Set(hunks.map((_, k) => k))); }} />}
    </div>
  </>;
}

// v10b · Project team: members made from archetypes, tuning, drift, promote (FR-10, FR-11, FR-12a, FR-12c, FR-12d, FR-12e)
export function ProjectTeam({ id, mid }: { id: string; mid?: string }) {
  const { data: members, error } = useData<any[]>(() => api.members.list(id), [id], (e) => /^(member|archetype)\./.test(e.type));
  const { data: archs } = useData<any[]>(() => api.archetypes.list(), []);
  const { data: roles } = useData<any[]>(() => api.roles.list(), []);
  const [adding, setAdding] = useState<'' | 'ai' | 'human'>('');
  const { run, busy } = useAction();
  const sel = mid || members?.find((m) => m.kind === 'ai')?.id || '';
  const roleName = (r: string) => (roles || []).find((x) => x.id === r)?.name || r;
  return <Shell title="Team" crumb={<A to={`/projects/${id}`}>{id}</A>} tabs={<ProjectTabs id={id} on="team" />}>
    <Err msg={error} />
    <div className="row">
      <Card style={{ width: 330, padding: '6px 10px' }}>
        <table className="t small"><tbody><tr><th>Member</th><th>From</th></tr>
          {(members || []).map((m) => <tr key={m.id} className={'click ' + (m.id === sel ? 'sel' : '')} {...clickable(() => navigate(`/projects/${id}/team/${encodeURIComponent(m.id)}`), { role: 'link' })}>
            <td><Av name={m.name} role={m.kind === 'human' ? 'human' : m.role} /> <b>{m.name}</b></td>
            <td>{m.kind === 'human' ? <span className="muted">human · {roleName(m.role)}</span> : <>{roleName(m.role)}{(m.tuning || m.harnessOverride || m.modelOverride) && <> · <Chip c="amb">tuned</Chip></>}{m.archetypeLatest > m.archetypeRev && <> · <Chip c="blue">update</Chip></>}</>}</td></tr>)}</tbody></table>
        {members && !members.length && <Empty>No team yet. <button type="button" className="lnk" onClick={() => run(() => api.members.starter(id), 'Starter team added')}>Add the starter team</button> (PM, architect, developer, reviewer).</Empty>}
        <div className="row c wrap" style={{ gap: 6, margin: '8px 0' }}>
          <button type="button" className="btn p" onClick={() => setAdding('ai')}>+ Member from archetype</button>
          <button type="button" className="btn" onClick={() => setAdding('human')}>+ Human</button></div>
        {adding && <AddMember projectId={id} kind={adding} archs={archs || []} roles={roles || []} busy={busy}
          onAdd={(body) => run(() => api.members.add(id, body), 'Member added').then((m: any) => { if (m) { setAdding(''); navigate(`/projects/${id}/team/${encodeURIComponent(m.id)}`); } })} onCancel={() => setAdding('')} />}
      </Card>
      {sel ? <MemberDetail key={sel} id={sel} projectId={id} /> : members?.length ? <Empty>Pick a member.</Empty> : null}
    </div>
  </Shell>;
}

function AddMember({ kind, archs, roles, busy, onAdd, onCancel }: { projectId: string; kind: 'ai' | 'human'; archs: any[]; roles: any[]; busy: boolean; onAdd: (b: any) => void; onCancel: () => void }) {
  const usable = archs.filter((a) => a.harness !== 'courier');
  const [arch, setArch] = useState(usable.find((a) => a.role === 'developer')?.id || usable[0]?.id || '');
  const [name, setName] = useState('');
  const [login, setLogin] = useState('');
  const [role, setRole] = useState('reviewer');
  return <div className="card" data-submit-scope="" style={{ padding: 10, marginTop: 6 }}>
    {kind === 'ai' ? <>
      <div className="form-row"><label>Archetype</label><select autoFocus value={arch} onChange={(e) => setArch(e.target.value)}>{usable.map((a) => <option key={a.id} value={a.id}>{a.name} · {a.role}</option>)}</select></div>
      <div className="form-row"><label>Name (optional, e.g. a second developer)</label><input value={name} onChange={(e) => setName(e.target.value)} placeholder={usable.find((a) => a.id === arch)?.name} /></div>
    </> : <>
      <div className="form-row"><label>Name</label><input autoFocus value={name} onChange={(e) => setName(e.target.value)} placeholder="maria" /></div>
      <div className="form-row"><label>Tailscale login (optional)</label><input value={login} onChange={(e) => setLogin(e.target.value)} placeholder="maria@github" /></div>
      <div className="form-row"><label>Role on this project</label><select value={role} onChange={(e) => setRole(e.target.value)}>{roles.map((r) => <option key={r.id} value={r.id}>{r.name}</option>)}</select></div>
    </>}
    <div className="row c"><span className="right" /><button type="button" className="btn" onClick={onCancel}>Cancel</button>
      <button type="button" className="btn p" disabled={busy || (kind === 'human' && !name)} onClick={() => onAdd(kind === 'ai' ? { kind, archetype: arch, name } : { kind, name, login, role })}>Add</button></div>
  </div>;
}

function MemberDetail({ id, projectId }: { id: string; projectId: string }) {
  const { data, reload } = useData<any>(() => api.members.get(id), [id], (e) => /^(member|archetype)\./.test(e.type) && (e.subject === id || e.type.startsWith('archetype')));
  const { data: harn } = useData<any[]>(() => api.harnesses.list(), []);
  const personas = usePersonas(projectId);
  const [t, setT] = useState<any>(null);
  const [view, setView] = useState<'tuning' | 'effective' | 'diff'>('tuning');
  const [mode, setMode] = useState<'wysiwyg' | 'markdown' | 'split'>('markdown');
  const [coach, setCoach] = useState('');
  const [prop, setProp] = useState<{ hunks: Hunk[]; base: string } | null>(null);
  const [acc, setAcc] = useState<Set<number>>(new Set());
  const [promote, setPromote] = useState('');
  const { run, busy } = useAction();
  const m = data?.member;
  useEffect(() => { if (m) setT({ name: m.name, role: m.role, tuning: m.tuning, replaceInstructions: m.replaceInstructions, harness: m.harnessOverride, model: m.modelOverride, login: m.login }); }, [m?.rev, m?.archetypeRev, m?.id]);
  useEffect(() => { if (!coach && personas.length) setCoach((personas.find((p) => p.role === 'architect') || personas[0]).id); }, [personas, coach]);
  const diff = useMemo(() => (data?.archetype && m ? lineDiff(data.archetype.instructions || '', m.instructions || '') : ''), [data?.archetype?.rev, m?.instructions]);
  if (!m || !t) return <Loading />;
  const save = () => run(() => api.members.tune(id, t), `Saved ${t.name}`).then(reload);
  const field = (label: string, el: React.ReactNode) => <label className="small" style={{ display: 'flex', gap: 6, alignItems: 'center' }}>{label} {el}</label>;
  if (m.kind === 'human') return <div className="col grow"><Card title={m.name}><div className="col small" style={{ gap: 8 }}>
    {field('Role on this project', <input value={t.role} onChange={(e) => setT({ ...t, role: e.target.value })} />)}
    {field('Tailscale login', <input value={t.login} onChange={(e) => setT({ ...t, login: e.target.value })} placeholder="name@github" />)}
    <span className="muted">Humans are global people; this sets their role on {projectId} (FR-12e).</span>
    <div className="row c"><button type="button" className="btn d" onClick={() => { if (window.confirm(`Remove ${m.name} from ${projectId}?`)) run(() => api.members.remove(id), 'Removed').then(() => navigate(`/projects/${projectId}/team`)); }}>Remove</button><span className="right" /><button type="button" className="btn p" disabled={busy} onClick={save}>Save</button></div></div></Card></div>;
  const behind = m.archetypeLatest > m.archetypeRev;
  return <>
    <div className="col grow">
      <Card style={{ padding: '10px 14px' }}>
        <div className="row c wrap" style={{ gap: 10 }}>
          <Av name={m.name} role={m.role} /><input aria-label="Name" value={t.name} onChange={(e) => setT({ ...t, name: e.target.value })} style={{ width: 120, fontWeight: 700 }} />
          <span className="small">from <A to={`/archetypes/${m.archetypeId}`}>{data.archetype?.name} rev {m.archetypeRev}</A>{behind && <> <Chip c="blue">rev {m.archetypeLatest} available</Chip></>}</span>
          {field('Harness', <select value={t.harness} onChange={(e) => setT({ ...t, harness: e.target.value })}><option value="">{data.archetype?.harness} (archetype)</option>{harnessOptions(harn).map((h) => <option key={h} value={h}>{h}</option>)}</select>)}
          {field('Model', <input value={t.model} onChange={(e) => setT({ ...t, model: e.target.value })} placeholder={data.archetype?.model || 'archetype default'} style={{ width: 130 }} />)}
        </div>
        <div className="small muted" style={{ marginTop: 6 }}>Worktree <span className="mono">{m.worktree || `~/.go-build-it/worktrees/${projectId}/${m.id.split('@')[0]}`}</span> · branch per task · herdr pane <span className="mono">{m.paneId || 'starts with the first task'}</span>
          {m.conversationId && <> · conversation <span className="mono">{m.conversationId.slice(0, 8)}…</span> ({m.conversationTasks} task{m.conversationTasks === 1 ? '' : 's'})</>}</div>
      </Card>
      {prop && <><Proposals hunks={prop.hunks} accepted={acc} setAccepted={setAcc} /><div className="row c"><span className="right" /><button type="button" className="btn" onClick={() => setProp(null)}>Discard</button>
        <button type="button" className="btn p" onClick={async () => { setT({ ...t, tuning: await applyHunks(prop.base, prop.hunks, acc) }); setProp(null); }}>Apply {acc.size} accepted</button></div></>}
      <div className="row c wrap" style={{ gap: 8 }}><Seg value={view} onChange={setView} options={[['tuning', 'Tuning'], ['effective', 'Effective'], ['diff', 'Diff vs archetype']]} />
        <label className="small" style={{ display: 'flex', gap: 6, alignItems: 'center' }}><input type="checkbox" checked={!!t.replaceInstructions} onChange={(e) => setT({ ...t, replaceInstructions: e.target.checked })} /> Replace the archetype's instructions</label></div>
      {view === 'tuning' && <Editor value={t.tuning} onChange={(v) => setT({ ...t, tuning: v })} mode={mode} onMode={setMode} toolbarRight={<span className="mono">{projectId}/team/{m.id.split('@')[0]}.md · tuning</span>} />}
      {view === 'effective' && <pre className="card mono small" style={{ whiteSpace: 'pre-wrap', margin: 0 }} tabIndex={0}>{data.persona}</pre>}
      {view === 'diff' && <div className="card" style={{ padding: 0 }}><DiffView text={diff} /></div>}
      <span className="small muted">Tuning is added under “## For this project” in the member's instruction file (AGENTS.md / CLAUDE.md in its worktree).</span>
      <div className="row c wrap" style={{ gap: 6 }}>
        <button type="button" className="btn d" onClick={() => { if (window.confirm(`Remove ${m.name} from ${projectId}?`)) run(() => api.members.remove(id), 'Removed').then(() => navigate(`/projects/${projectId}/team`)); }}>Remove</button>
        <span className="right" />
        {behind && <button type="button" className="btn" disabled={busy} onClick={() => run(() => api.archetypes.apply(m.archetypeId, { member: id }), `Moved to rev ${m.archetypeLatest}`).then(reload)}>Apply archetype rev {m.archetypeLatest}</button>}
        <button type="button" className="btn p" disabled={busy} onClick={save}>Save tuning</button></div>
      {(m.tuning || '').trim() && <Card title="Promote to the archetype"><div className="col small" style={{ gap: 6 }}>
        <span className="muted">Good tuning can become part of {data.archetype?.name} for every project (FR-12c). Pick lines:</span>
        <div className="col" style={{ gap: 2 }}>{(m.tuning as string).split('\n').filter((l) => l.trim()).map((l, k) => <label key={k} className="mono" style={{ display: 'flex', gap: 6 }}><input type="checkbox" checked={promote.split('\n').includes(l)} onChange={(e) => setPromote(e.target.checked ? [promote, l].filter(Boolean).join('\n') : promote.split('\n').filter((x) => x !== l).join('\n'))} />{l}</label>)}</div>
        <div className="row c"><span className="right" /><button type="button" className="btn" disabled={busy || !promote} onClick={() => run(() => api.members.promote(id, { text: promote }), 'Promoted').then(() => { setPromote(''); reload(); })}>Promote {promote ? promote.split('\n').length : 0} line(s)</button></div>
      </div></Card>}
    </div>
    <div style={{ width: 300 }}><AgentChat title="Persona coach" personas={personas} persona={coach} setPersona={setCoach} placeholder={`How should ${m.name} work on ${projectId}?`}
      suggestions={['Interview me', 'Check against standards', 'Learn from history']}
      send={(msg, h) => api.members.chat(id, { coach, message: msg, content: t.tuning || '', history: h })}
      onProposal={(_d, hunks) => { setProp({ hunks, base: t.tuning || '' }); setAcc(new Set(hunks.map((_, k) => k))); }} /></div>
  </>;
}
