// Learn page (view 18) and a lesson's full text (FR-134–FR-138, ADR-012).
import { api } from '../api.gen';
import { LessonSteps, PracticeCard, useLearn, useLesson, useLessons, useThinking, type Lesson } from '../learn';
import { Shell } from '../shell';
import { A, Card, Chip, Err, Loading, Markdown, useAction } from '../ui';

function LessonCard({ l, onOpen }: { l: Lesson; onOpen: (l: Lesson) => void }) {
  const done = l.done === l.total && l.total > 0;
  const going = !!l.started && !done;
  return <div className={'card lesson-card' + (going ? ' going' : '')} data-lesson={l.slug} data-done={done ? 'true' : 'false'}>
    <h4>{done ? '✅ ' : going ? '▶ ' : ''}{l.number} · {l.title}</h4>
    <div className="small muted">{l.minutes} min · {l.summary}</div>
    {going && <div className="bar"><i style={{ width: `${(l.done / l.total) * 100}%` }} /></div>}
    <div className="row small" style={{ marginTop: 6, gap: 8 }}>
      <button type="button" className={'btn sm' + (done ? '' : ' p')} onClick={() => onOpen(l)} data-nav="">{done ? 'Again' : going ? `Continue · step ${l.steps.findIndex((s) => !s.done) + 1} of ${l.total}` : 'Start'}</button>
      <A to={'/learn/' + l.slug}>Read</A>
      {l.desktopOnly && <Chip>desktop</Chip>}
    </div>
  </div>;
}

export function Learn() {
  const { data, error, reload } = useLessons();
  const { data: docs } = useThinking();
  const { open, hide } = useLearn();
  const { run } = useAction();
  const start = async (l: Lesson) => {
    if (!l.started || l.done === l.total) await run(() => api.learn.start(l.slug));
    open(l.slug); hide(false);
  };
  const ls = data?.lessons || [];
  const done = ls.filter((l) => l.total > 0 && l.done === l.total).length;
  const thinking = (docs || []).filter((p: any) => p.section === 'How go-build-it thinks');
  return <Shell title="Learn" right={data && <span className="muted small" data-testid="learn-progress">{done} of {ls.length} lessons done</span>}>
    <Err msg={error} />
    {!data ? <Loading what="the lessons" /> : <>
      <PracticeCard practice={data.practice} reload={reload} />
      {!data.practice && <div className="small muted">Lessons use the practice project; create it first (it takes a second).</div>}
      <div className="lessons" role="list" aria-label="Lessons">{ls.map((l) => <div role="listitem" key={l.slug}><LessonCard l={l} onOpen={start} /></div>)}</div>
      <Card title="How go-build-it thinks"><div className="small thinking">
        {thinking.length === 0 ? <span className="muted">…</span> : thinking.map((p: any) => <div key={p.slug}>• <A to={'/docs/' + p.slug}>{p.title}</A></div>)}
      </div></Card>
    </>}
  </Shell>;
}

export function LessonPage({ slug }: { slug: string }) {
  const { data, error, setData } = useLesson(slug);
  const { open, hide } = useLearn();
  const { run } = useAction();
  const l = data?.lesson;
  const begin = async () => {
    if (!l) return;
    if (!l.started) { const d: any = await run(() => api.learn.start(l.slug)); if (d) setData(d); }
    open(l.slug); hide(false);
  };
  return <Shell title={l ? `${l.number} · ${l.title}` : 'Lesson'} crumb={<A to="/learn">Learn</A>} right={l && <button type="button" className="btn p sm" onClick={begin} data-testid="open-coach">{l.started ? 'Continue in the coach' : 'Start this lesson'}</button>}>
    <Err msg={error} />
    {!l ? <Loading what="the lesson" /> : <Card title={`${l.minutes} min · ${l.done} of ${l.total} steps done`}>
      <Markdown src={l.intro} />
      {(l.tips || []).map((t, i) => <div className="tip" key={i}><b>Pro tip:</b> <Markdown src={t} className="inline-md" /></div>)}
      <div className="lesson-full"><LessonSteps full lesson={l} practice={data!.practice} onDone={(n) => run(() => api.learn.done(l.slug, { step: n })).then((d: any) => d && setData(d))} /></div>
      {l.related && l.related.length > 0 && <div className="small muted" style={{ marginTop: 8 }}>Related: {l.related.map((r, i) => <span key={r}>{i ? ' · ' : ''}<A to={'/docs/' + r}>{r}</A></span>)}</div>}
    </Card>}
  </Shell>;
}
