import { match, usePath } from './router';
import { SystemProvider } from './shell';
import { ToastProvider } from './ui';
import { Catalog, MilestoneDetail, ProjectHistory, ProjectOverview, ProjectReleases, ProjectSettings } from './pages/Projects';
import { ArtifactEditor, Artifacts } from './pages/Artifacts';
import { Queue, Questions, SessionDetail, Sessions } from './pages/Work';
import { OptionSets, Rehearsals, Reviews } from './pages/Review';
import { Docs, EventDetail, History, Settings, Standards, Team } from './pages/Org';
import { MobileApp } from './mobile/Mobile';

type R = [string, (p: Record<string, string>) => JSX.Element];
const routes: R[] = [
  ['/', () => <Catalog />],
  ['/projects/:id', (p) => <ProjectOverview id={p.id} />],
  ['/projects/:id/artifacts', (p) => <Artifacts id={p.id} />],
  ['/projects/:id/edit', (p) => <ArtifactEditor id={p.id} />],
  ['/projects/:id/queue', (p) => <Queue projectId={p.id} />],
  ['/projects/:id/sessions', (p) => <Sessions projectId={p.id} />],
  ['/projects/:id/releases', (p) => <ProjectReleases id={p.id} />],
  ['/projects/:id/history', (p) => <ProjectHistory id={p.id} />],
  ['/projects/:id/settings', (p) => <ProjectSettings id={p.id} />],
  ['/milestones/:mid', (p) => <MilestoneDetail mid={p.mid} />],
  ['/questions', () => <Questions />],
  ['/questions/:qid', (p) => <Questions qid={p.qid} />],
  ['/sessions', () => <Sessions />],
  ['/sessions/:sid', (p) => <SessionDetail sid={p.sid} />],
  ['/queue', () => <Queue />],
  ['/reviews', () => <Reviews />],
  ['/reviews/:rid', (p) => <Reviews rid={p.rid} />],
  ['/options', () => <OptionSets />],
  ['/options/:oid', (p) => <OptionSets oid={p.oid} />],
  ['/rehearsals', () => <Rehearsals />],
  ['/rehearsals/:gid', (p) => <Rehearsals gid={p.gid} />],
  ['/team', () => <Team />],
  ['/team/:mid', (p) => <Team mid={p.mid} />],
  ['/standards', () => <Standards />],
  ['/standards/:sid', (p) => <Standards sid={p.sid} />],
  ['/history', () => <History />],
  ['/history/:eid', (p) => <EventDetail eid={p.eid} />],
  ['/settings', () => <Settings />],
  ['/docs', () => <Docs />],
  ['/docs/:slug', (p) => <Docs slug={p.slug} />],
];

export function App() {
  const path = usePath();
  let el: JSX.Element | null = null;
  if (path === '/m' || path.startsWith('/m/') || path.startsWith('/m?')) el = <MobileApp />;
  else for (const [pat, render] of routes) { const m = match(pat, path); if (m) { el = render(m); break; } }
  return <ToastProvider><SystemProvider>{el || <div className="empty">Page not found. <a href="/">Projects</a></div>}</SystemProvider></ToastProvider>;
}
