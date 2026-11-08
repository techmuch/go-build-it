// Live updates over Server-Sent Events (FR-36).
import { useEffect, useRef, useState, useCallback } from 'react';

export type LiveEvent = { id: number; ts: string; type: string; actor: string; projectId: string; subject: string; summary: string; why: string; data: any };
type Listener = (e: LiveEvent) => void;

const listeners = new Set<Listener>();
let source: EventSource | null = null;
let connected = false;
const statusListeners = new Set<(c: boolean) => void>();

function ensure() {
  if (source) return;
  source = new EventSource('/api/events/stream');
  source.onopen = () => { connected = true; statusListeners.forEach((l) => l(true)); };
  source.onerror = () => { connected = false; statusListeners.forEach((l) => l(false)); };
  source.onmessage = (m) => dispatch(m.data);
}

function dispatch(raw: string) {
  try { const e = JSON.parse(raw) as LiveEvent; listeners.forEach((l) => l(e)); } catch { /* ignore */ }
}

export function useLive(fn: Listener) {
  const ref = useRef(fn);
  ref.current = fn;
  useEffect(() => {
    ensure();
    const l: Listener = (e) => ref.current(e);
    listeners.add(l);
    return () => { listeners.delete(l); };
  }, []);
}

export function useConnected() {
  const [c, setC] = useState(connected);
  useEffect(() => { ensure(); statusListeners.add(setC); return () => { statusListeners.delete(setC); }; }, []);
  return c;
}

// useData loads data and reloads when a matching live event arrives (debounced).
export function useData<T>(load: () => Promise<T>, deps: unknown[], match?: (e: LiveEvent) => boolean) {
  const [data, setData] = useState<T | undefined>(undefined);
  const [error, setError] = useState<string>('');
  const [loading, setLoading] = useState(true);
  const timer = useRef<number | undefined>(undefined);
  const reload = useCallback(() => {
    load().then((d) => { setData(d); setError(''); }).catch((e) => setError(String(e.message || e))).finally(() => setLoading(false));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps);
  useEffect(() => { setLoading(true); reload(); }, [reload]);
  useLive((e) => {
    if (e.type === 'session.output') return;
    if (match && !match(e)) return;
    window.clearTimeout(timer.current);
    timer.current = window.setTimeout(reload, 250);
  });
  return { data, error, loading, reload, setData };
}
