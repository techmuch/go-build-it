// Phones open the mobile UI (FR-35a): a desktop address opened on a phone goes to the matching
// /m/ screen, unless this device chose "Desktop site". The choice is per device (localStorage).
import { match } from './router';

const KEY = 'gbi.desktopSite';

export function wantsDesktop(): boolean {
  try { return localStorage.getItem(KEY) === '1'; } catch { return false; }
}

export function setWantsDesktop(on: boolean) {
  try { if (on) localStorage.setItem(KEY, '1'); else localStorage.removeItem(KEY); } catch { /* private window: no memory */ }
}

// A phone: a narrow screen with a touch pointer. A narrow desktop window keeps the desktop UI.
export function isPhone(): boolean {
  try { return window.matchMedia('(max-width: 760px) and (pointer: coarse)').matches; } catch { return false; }
}

const toMobile: [string, (p: Record<string, string>) => string][] = [
  ['/', () => '/m/'],
  ['/projects/:id/artifacts', (p) => `/m/projects/${p.id}/docs`],
  ['/projects/:id/artifacts/:tab/:doc*', (p) => `/m/projects/${p.id}/docs/${p.tab}`],
  ['/projects/:id/:rest*', (p) => `/m/projects/${p.id}`],
  ['/projects/:id', (p) => `/m/projects/${p.id}`],
  ['/questions/:qid', (p) => `/m/q/${p.qid}`],
  ['/reviews/:rid', (p) => `/m/review/${p.rid}`],
  ['/rehearsals/:gid', (p) => `/m/rehearsal/${p.gid}`],
  ['/options/:oid', (p) => `/m/options/${p.oid}`],
  ['/questions', () => '/m/inbox'], ['/reviews', () => '/m/inbox'], ['/rehearsals', () => '/m/inbox'], ['/options', () => '/m/inbox'],
  ['/sessions/:sid', (p) => `/m/sessions/${p.sid}`],
  ['/sessions', () => '/m/sessions'],
  ['/changelog', () => '/m/changelog'],
  ['/learn', () => '/m/learn'], ['/learn/:slug', () => '/m/learn'],
  ['/settings', () => '/m/more'],
];

// mobilePath is the mobile screen for a desktop address; screens without one go to /m/.
export function mobilePath(path: string): string {
  const [p] = path.split('?');
  for (const [pat, to] of toMobile) { const m = match(pat, p); if (m) return to(m); }
  return '/m/';
}
