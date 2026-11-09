// Change log and guided updates (FR-06a–FR-06e, ADR-011). Opened from the version number.
// The update panel checks for newer releases, starts an update, and shows its steps as they
// complete, polling through the restart; the change log below lists what each version changed.
import { useCallback, useEffect, useRef, useState } from 'react';
import { api } from '../api.gen';
import { useData } from '../live';
import { Shell, useSystem } from '../shell';
import { Card, Chip, Err, Loading, Markdown, ago, day } from '../ui';

type Step = { id: string; title: string; status: 'pending' | 'running' | 'done' | 'failed' | 'skipped'; detail?: string; error?: string; startedAt?: string; endedAt?: string; bytes?: number; total?: number };
export type UpdateJob = { id: string; kind: 'update' | 'rollback'; trigger: string; actor?: string; from: string; to: string; status: 'running' | 'restarting' | 'done' | 'failed'; error?: string; startedAt: string; endedAt?: string; steps: Step[]; canRetry: boolean; canRollback: boolean };
type Check = { current: string; tag?: string; newer?: boolean; error?: string; previous?: string; checkedAt?: string; releases: { tag: string; date: string; notes: string; url: string }[] };
type Entry = { version: string; date?: string; body: string };

const active = (j?: UpdateJob | null) => !!j && (j.status === 'running' || j.status === 'restarting');
const secs = (s: Step) => (s.startedAt && s.endedAt ? ((new Date(s.endedAt).getTime() - new Date(s.startedAt).getTime()) / 1000).toFixed(1) + 's' : '');
const ICON: Record<Step['status'], string> = { done: '✅', running: '⏳', pending: '○', failed: '✖', skipped: '–' };

// useUpdateJob follows the running or last update. While one runs it polls twice a second;
// when the server is away (restarting) it keeps polling and reports offline (ADR-011).
export function useUpdateJob() {
  const [job, setJob] = useState<UpdateJob | null | undefined>(undefined);
  const [offline, setOffline] = useState(false);
  const jobRef = useRef(job);
  jobRef.current = job;
  const poll = useCallback(async () => {
    try {
      const r: any = await api.updates.job();
      setJob(r.job || null);
      setOffline(false);
    } catch {
      if (active(jobRef.current)) setOffline(true);
    }
  }, []);
  useEffect(() => { poll(); }, [poll]);
  useEffect(() => {
    if (!active(job) && !offline) return;
    const t = window.setInterval(poll, 500);
    return () => window.clearInterval(t);
  }, [job, offline, poll]);
  return { job, setJob, offline, poll };
}

export function Steps({ job, offline }: { job: UpdateJob; offline: boolean }) {
  const done = job.steps.filter((s) => s.status === 'done').length;
  const at = job.steps.findIndex((s) => s.status === 'running');
  return <div className="usteps">
    <div className="small muted" style={{ marginBottom: 4 }}>{job.status === 'done' ? `All ${job.steps.length} steps done` : job.status === 'failed' ? `Stopped at step ${job.steps.findIndex((s) => s.status === 'failed') + 1} of ${job.steps.length}` : `Step ${(at < 0 ? done : at) + 1} of ${job.steps.length}`}</div>
    <div className="bar" role="progressbar" aria-label="Update progress" aria-valuemin={0} aria-valuemax={job.steps.length} aria-valuenow={done}><i style={{ width: `${(done / job.steps.length) * 100}%` }} /></div>
    <ol className="usteplist" aria-live="polite" aria-label="Update steps">
      {job.steps.map((s) => <li key={s.id} className={'ustep ' + s.status} data-step={s.id} data-status={s.status}>
        <span className="ui" aria-hidden="true">{ICON[s.status]}</span>
        <span className="ut"><b>{s.title}</b> <span className="sr-only">({s.status})</span><br />
          <span className="muted">{s.id === 'restart' && s.status === 'running' && offline ? 'Waiting for go-build-it to come back…' : s.detail}</span>
          {s.id === 'download' && s.status === 'running' && !!s.total && <span className="bar thin"><i style={{ width: `${Math.min(100, ((s.bytes || 0) / s.total) * 100)}%` }} /></span>}
          {s.error && <span className="err small" style={{ display: 'block', marginTop: 4 }}>{s.error}</span>}</span>
        <span className="uz muted">{secs(s)}</span>
      </li>)}
    </ol>
  </div>;
}

// UpdatePanel: check, Update now, progress, Try again, Roll back (FR-06c, FR-06d).
export function UpdatePanel({ onCheck }: { onCheck?: (c: Check) => void }) {
  const sys = useSystem();
  const { job, setJob, offline, poll } = useUpdateJob();
  const [check, setCheck] = useState<Check | undefined>(undefined);
  const [checking, setChecking] = useState(false);
  const [err, setErr] = useState('');
  const loadedVersion = useRef<string | undefined>(undefined);
  if (sys && !loadedVersion.current) loadedVersion.current = sys.version;

  const runCheck = useCallback(async () => {
    setChecking(true); setErr('');
    try { const c = await api.updates.check(); setCheck(c); onCheck?.(c); } catch (e: any) { setErr(String(e.message || e)); } finally { setChecking(false); }
  }, [onCheck]);
  useEffect(() => { runCheck(); }, [runCheck]);

  // The new binary serves a new UI: once the update is done, load it.
  const reloading = job?.status === 'done' && !!loadedVersion.current && !!job.to && job.to.replace(/^v/, '') !== loadedVersion.current.replace(/^v/, '') && Date.now() - new Date(job.endedAt || 0).getTime() < 120_000;
  useEffect(() => { if (reloading) { const t = window.setTimeout(() => window.location.reload(), 1500); return () => window.clearTimeout(t); } }, [reloading]);

  const start = async (rollback = false) => {
    setErr('');
    try { const r: any = rollback ? await api.updates.rollback() : await api.updates.install(); setJob(r.job); } catch (e: any) { setErr(String(e.message || e)); poll(); }
  };
  const busy = active(job);
  const n = check?.releases?.length || 0;
  const previous = check?.previous;
  return <div className="upanel" aria-busy={busy}>
    <div className="small" style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
      <div>Installed <b data-testid="installed-version">{sys?.version || '…'}</b>{sys?.version === 'dev' && <span className="muted"> · development build</span>}</div>
      {check && !check.error && <div>Latest <b data-testid="latest-version">{check.tag}</b> {check.newer ? <Chip c="blue">{n === 1 ? '1 release newer' : `${n} releases newer`}</Chip> : <Chip c="grn">up to date</Chip>}</div>}
      {check?.error && <div className="muted" data-testid="check-error">Could not check: {check.error}</div>}
      <div className="muted">{checking ? 'Checking…' : check?.checkedAt ? <>Checked {ago(check.checkedAt)}</> : null} · <button type="button" className="lnk" onClick={runCheck} disabled={checking || busy}>Check again</button></div>
    </div>
    <Err msg={err} />
    {job && (busy || job.status === 'failed' || (job.status === 'done' && Date.now() - new Date(job.endedAt || 0).getTime() < 10 * 60_000)) && <div className="ujob" data-testid="update-job" data-status={job.status}>
      <h4>{job.status === 'done' ? <>✅ {job.kind === 'rollback' ? 'Rolled back to' : 'Updated to'} {job.to}</> : job.status === 'failed' ? <>{job.kind === 'rollback' ? 'Rollback' : 'Update'} {job.to ? `to ${job.to}` : ''} failed</> : <>{job.kind === 'rollback' ? 'Rolling back to' : 'Updating to'} {job.to || 'the latest release'}</>}
        {job.trigger === 'auto' && <Chip>automatic</Chip>}</h4>
      <Steps job={job} offline={offline} />
      {reloading && <div className="small muted">Loading the new version's page…</div>}
      {job.status === 'failed' && <div className="row" style={{ marginTop: 8 }}>
        {job.canRetry && <button type="button" className="btn p sm" onClick={() => start()}>Try again</button>}
        {job.canRollback && <button type="button" className="btn sm" onClick={() => start(true)}>Roll back to {job.from}</button>}
      </div>}
      {busy && <div className="small muted" style={{ marginTop: 6 }}>You can leave this page; the update keeps going and is recorded in the history.</div>}
    </div>}
    {!busy && check?.newer && !(job?.status === 'failed' && job.canRetry) && <div style={{ marginTop: 10 }}>
      {!!sys?.liveSessions && <div className="small note" style={{ marginBottom: 8 }}>{sys.liveSessions} {sys.liveSessions === 1 ? 'member is' : 'members are'} working. They keep running in herdr while go-build-it restarts.</div>}
      <button type="button" className="btn p" onClick={() => start()} data-testid="update-now">⬆ Update now to {check.tag}</button>
    </div>}
    {!busy && previous && <div className="small muted" style={{ marginTop: 10 }}>Previous: {previous} · <button type="button" className="lnk" onClick={() => start(true)}>Roll back</button></div>}
  </div>;
}

// ChangelogList: newer releases (from the check) first, then the embedded change log.
export function ChangelogList({ check }: { check?: Check }) {
  const sys = useSystem();
  const { data, error } = useData<{ current: string; entries: Entry[] }>(() => api.changelog.list(), []);
  if (error) return <Err msg={error} />;
  if (!data) return <Loading what="the change log" />;
  const cur = data.current;
  const installed = (v: string) => (cur === 'dev' ? v === 'Unreleased' : v.replace(/^v/, '') === cur.replace(/^v/, ''));
  const known = new Set(data.entries.map((e) => e.version));
  const newer = (check?.releases || []).filter((r) => !known.has(r.tag.replace(/^v/, '')));
  return <div className="clog">
    {newer.map((r) => <article key={r.tag} className="centry" aria-label={`Release ${r.tag}`}>
      <h3>{r.tag} <Chip c="blue">new</Chip> {r.date && <span className="muted small">{day(r.date + 'T12:00:00')}</span>}</h3>
      <Markdown src={r.notes || '_No notes._'} />
    </article>)}
    {data.entries.map((e) => <article key={e.version} className={'centry' + (installed(e.version) ? ' installed' : '')} aria-label={`Release ${e.version}`} aria-current={installed(e.version) ? 'true' : undefined}>
      <h3>{e.version}{installed(e.version) && <> <Chip c="grn">installed{sys?.version === 'dev' ? ' (dev build)' : ''}</Chip></>} {e.date && <span className="muted small">{day(e.date + 'T12:00:00')}</span>}</h3>
      {e.body ? <Markdown src={e.body} /> : <p className="muted small">Nothing yet.</p>}
    </article>)}
  </div>;
}

// Desktop page: /changelog (g v).
export function Changelog() {
  const sys = useSystem();
  const [check, setCheck] = useState<Check | undefined>(undefined);
  return <Shell title="Change log" right={<span className="muted small">{sys ? `v${sys.version} installed` : ''}</span>}>
    <div className="grid changelog-grid" style={{ gridTemplateColumns: 'minmax(0,1fr) 360px', alignItems: 'start' }}>
      <Card title="What changed" right={<span className="muted small">from CHANGELOG.md · newest first</span>}><ChangelogList check={check} /></Card>
      <Card title="Updates"><UpdatePanel onCheck={setCheck} /></Card>
    </div>
  </Shell>;
}
