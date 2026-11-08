// Minimal history router.
import { useEffect, useState } from 'react';

export function navigate(to: string) {
  if (to === location.pathname + location.search) return;
  history.pushState(null, '', to);
  window.dispatchEvent(new Event('gbi:navigate'));
  window.scrollTo(0, 0);
}

export function usePath() {
  const [p, setP] = useState(location.pathname + location.search);
  useEffect(() => {
    const on = () => setP(location.pathname + location.search);
    window.addEventListener('popstate', on);
    window.addEventListener('gbi:navigate', on);
    return () => { window.removeEventListener('popstate', on); window.removeEventListener('gbi:navigate', on); };
  }, []);
  return p;
}

export function match(pattern: string, path: string): Record<string, string> | null {
  const [pp] = path.split('?');
  const a = pattern.split('/').filter(Boolean);
  const b = pp.split('/').filter(Boolean);
  const out: Record<string, string> = {};
  for (let i = 0; i < a.length; i++) {
    if (a[i].endsWith('*')) { out[a[i].slice(1, -1)] = decodeURIComponent(b.slice(i).join('/')); return out; }
    if (i >= b.length) return null;
    if (a[i].startsWith(':')) out[a[i].slice(1)] = decodeURIComponent(b[i]);
    else if (a[i] !== b[i]) return null;
  }
  return a.length === b.length ? out : null;
}

export function query(name: string): string {
  return new URLSearchParams(location.search).get(name) || '';
}
