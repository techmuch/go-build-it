// Learn: lessons, the coach panel and the practice project (FR-134–FR-138, ADR-012; views 18, 19).
// A lesson opens as a coach docked beside the app. Its steps tick themselves off from the
// history (the server checks events in the practice project) and from the views you open.
import { createContext, useCallback, useContext, useEffect, useRef, useState, type ReactNode } from 'react';
import { api } from './api.gen';
import { useData } from './live';
import { navigate, usePath } from './router';
import { A, Chip, Err, Loading, Markdown, useAction } from './ui';

export type Check = { kind: 'event' | 'visit'; arg: string; any?: boolean };
export type Step = { n: number; title: string; body: string; checks?: Check[]; show?: { path: string; selector?: string }; tips?: string[]; done: boolean; how?: string };
export type Lesson = { slug: string; number: number; title: string; minutes: number; summary: string; desktopOnly?: boolean; intro: string; tips?: string[]; steps: Step[]; related?: string[]; started?: string; done: number; total: number };
export type Practice = { id: string; name: string; path: string } | null;

const KEY = 'gbi.learn.active';
const read = (): { slug?: string; hidden?: boolean } => { try { return JSON.parse(localStorage.getItem(KEY) || '{}'); } catch { return {}; } };
const write = (v: { slug?: string; hidden?: boolean }) => { try { localStorage.setItem(KEY, JSON.stringify(v)); } catch { /* private window */ } };

type Ctx = { active?: string; hidden: boolean; seq: number; open: (slug: string) => void; close: () => void; hide: (h: boolean) => void };
const LearnCtx = createContext<Ctx>({ hidden: true, seq: 0, open: () => {}, close: () => {}, hide: () => {} });
export function useLearn() { return useContext(LearnCtx); }

export const practicePath = (p: string, practice?: Practice) => p.replace(/\{practice\}/g, practice?.id || 'practice');
export const stepLabel = (c: Check) => c.kind === 'visit' ? `opening ${c.arg.replace('{practice}', 'practice')}` : `${c.arg.replace('*', '…')} ${c.any ? '(any project)' : 'in the practice project'}`;

// LearnProvider remembers the open lesson and reports the views you open (visit steps).
export function LearnProvider({ children }: { children: ReactNode }) {
  const [st, setSt] = useState(read);
  const path = usePath();
  const set = useCallback((f: (v: { slug?: string; hidden?: boolean }) => { slug?: string; hidden?: boolean }) => setSt((v) => { const n = f(v); write(n); return n; }), []);
  const [seq, setSeq] = useState(0); // bumped when a visit checks a step off (the live stream may not be connected yet)
  useEffect(() => {
    if (!st.slug) return;
    api.learn.visit({ path }).then((r: any) => { if (r?.marked) setSeq((n) => n + 1); }).catch(() => {});
  }, [path, st.slug]);
  return <LearnCtx.Provider value={{ active: st.slug, hidden: !!st.hidden, seq, open: (slug) => set(() => ({ slug })), close: () => set(() => ({})), hide: (h) => set((v) => ({ ...v, hidden: h })) }}>{children}</LearnCtx.Provider>;
}

// showMe goes to a step's view and pulses the control it names (FR-134).
export function showMe(step: Step, practice: Practice, mobile = false) {
  if (!step.show) return;
  let to = practicePath(step.show.path, practice);
  if (mobile && !to.startsWith('/m')) return;
  navigate(to);
  const sel = step.show.selector;
  const started = Date.now();
  const find = () => {
    const el = (sel && document.querySelector<HTMLElement>(sel)) || (Date.now() - started > 1500 ? document.querySelector<HTMLElement>('#main h1, .m-hdr h1') : null);
    if (!el) { if (Date.now() - started < 3000) window.setTimeout(find, 100); return; }
    el.scrollIntoView({ block: 'center', behavior: 'smooth' });
    el.classList.add('tour-pulse');
    el.setAttribute('data-tour-pulse', '1');
    window.setTimeout(() => { el.classList.remove('tour-pulse'); el.removeAttribute('data-tour-pulse'); }, 4000);
  };
  window.setTimeout(find, 50);
}

export function useLesson(slug?: string) {
  return useData<{ lesson: Lesson; practice: Practice }>(() => (slug ? api.learn.get(slug) : Promise.resolve(undefined as any)), [slug], (e) => !!slug && !e.type.startsWith('session.usage'));
}

function Tip({ text }: { text: string }) {
  return <div className="tip"><b>Pro tip:</b> <Markdown src={text} className="inline-md" /></div>;
}

export function StepView({ s, practice, current, onDone, mobile, full }: { s: Step; practice: Practice; current: boolean; onDone: (n: number) => void; mobile?: boolean; full?: boolean }) {
  const checks = s.checks || [];
  return <li className={'lstep ' + (s.done ? 'done' : current ? 'cur' : 'todo')} data-step={s.n} data-done={s.done ? 'true' : 'false'}>
    <span className="ln" aria-hidden="true">{s.done ? '✅' : s.n}</span>
    <div className="lb">
      <b>{s.title}</b> <span className="sr-only">{s.done ? '(done)' : current ? '(current)' : ''}</span>
      {(current || full) && s.body && <Markdown src={practicePath(s.body, practice)} />}
      {current && <div className="row" style={{ gap: 6, flexWrap: 'wrap', marginTop: 4 }}>
        {s.show && (!mobile || s.show.path.startsWith('/m')) && <button type="button" className="btn sm" onClick={() => showMe(s, practice, mobile)} data-testid="show-me">{mobile ? 'Go there' : 'Show me'}</button>}
        {!s.done && <button type="button" className="lnk small" onClick={() => onDone(s.n)} data-testid="mark-done">Mark done</button>}
      </div>}
      {current && !s.done && checks.length > 0 && <div className="small muted" style={{ marginTop: 4 }}>Waiting for: {checks.map(stepLabel).join(' or ')}</div>}
      {(current || full) && (s.tips || []).map((t, i) => <Tip key={i} text={t} />)}
      {mobile && s.show && !s.show.path.startsWith('/m') && current && <div className="small muted">This step is on the desktop UI.</div>}
    </div>
  </li>;
}

export function LessonSteps({ lesson, practice, onDone, mobile, full }: { lesson: Lesson; practice: Practice; onDone: (n: number) => void; mobile?: boolean; full?: boolean }) {
  const cur = lesson.steps.find((s) => !s.done)?.n;
  return <ol className="lsteps" aria-label={`Steps of lesson ${lesson.number}`} aria-live="polite">
    {lesson.steps.map((s) => <StepView key={s.n} s={s} practice={practice} current={s.n === cur} onDone={onDone} mobile={mobile} full={full} />)}
  </ol>;
}

// Coach: the active lesson, docked beside every page (view 19).
export function Coach() {
  const { active, hidden, close, hide, seq } = useLearn();
  const { data, error, setData, reload } = useLesson(active && !hidden ? active : undefined);
  useEffect(() => { if (seq) reload(); }, [seq, reload]);
  const { run } = useAction();
  const ref = useRef<HTMLElement>(null);
  const done = useCallback((n: number) => { if (active) run(() => api.learn.done(active, { step: n })).then((d: any) => d && setData(d)); }, [active, run, setData]);
  if (!active || hidden) return null;
  const l = data?.lesson;
  const finished = l && l.done === l.total;
  return <aside className="coach" aria-label="Lesson coach" ref={ref} data-testid="coach" onKeyDown={(e) => { if (e.key === 'Escape') { e.stopPropagation(); hide(true); } }}>
    <div className="row" style={{ alignItems: 'center' }}><b>{l ? `Lesson ${l.number} · ${l.title}` : 'Lesson'}</b><span className="right" />
      <button type="button" className="lnk small" onClick={() => hide(true)} title="Hide (Esc); g l brings it back" aria-label="Hide the coach">✕</button></div>
    <Err msg={error} />
    {!l ? <Loading what="the lesson" /> : <>
      <div className="bar" role="progressbar" aria-label="Lesson progress" aria-valuemin={0} aria-valuemax={l.total} aria-valuenow={l.done}><i style={{ width: `${(l.done / l.total) * 100}%` }} /></div>
      {!l.started && <div className="small note">Steps check themselves off once you start. <button type="button" className="btn p sm" onClick={() => run(() => api.learn.start(l.slug)).then((d: any) => d && setData(d))} data-testid="coach-start">Start</button></div>}
      <LessonSteps lesson={l} practice={data!.practice} onDone={done} />
      {(l.tips || []).map((t, i) => <Tip key={i} text={t} />)}
      {finished && <div className="small note" data-testid="lesson-finished">✅ Lesson done. <A to="/learn">Next lesson ›</A></div>}
      <div className="small muted"><A to={'/learn/' + l.slug}>Full lesson</A> · <button type="button" className="lnk" onClick={() => run(() => api.learn.start(l.slug)).then((d: any) => d && setData(d))}>Restart</button> · <button type="button" className="lnk" onClick={close}>Close lesson</button></div>
    </>}
  </aside>;
}

// PracticeCard creates, opens, resets or removes the practice project (FR-136).
export function PracticeCard({ practice, reload }: { practice: Practice; reload: () => void }) {
  const { run, busy } = useAction();
  const act = (o: { reset?: boolean; remove?: boolean }, msg: string) => run(() => api.learn.practice(o), msg).then(reload);
  return <div className="card practice-card" data-testid="practice-card">
    <div style={{ flex: 1, minWidth: 0 }}>
      <b>Practice project</b> <Chip c="amb">practice</Chip>
      <div className="small muted">{practice ? <>{practice.name} · <span className="mono">{practice.path}</span> · its AI members use the scripted harness, so no AI subscription is used</> : 'A small project to practise in. Its AI members are scripted: nothing here uses an AI subscription or touches your repositories.'}</div>
    </div>
    {practice ? <>
      <A to={'/projects/' + practice.id} className="btn sm">Open</A>
      <button type="button" className="btn sm" disabled={busy} onClick={() => window.confirm('Reset the practice project? Its tasks, members and files are recreated.') && act({ reset: true }, 'Practice project reset')}>Reset</button>
      <button type="button" className="btn sm" disabled={busy} onClick={() => window.confirm('Remove the practice project and its files?') && act({ remove: true }, 'Practice project removed')}>Remove</button>
    </> : <button type="button" className="btn p sm" disabled={busy} onClick={() => act({}, 'Practice project created')} data-testid="create-practice">Create the practice project</button>}
  </div>;
}

export function useLessons() {
  return useData<{ lessons: Lesson[]; practice: Practice }>(() => api.learn.list(), [], (e) => e.type.startsWith('learn') || e.type.startsWith('project'));
}

export function useThinking() {
  return useData<any[]>(() => api.docs.list(), []);
}
