// Live terminal of a member's herdr pane (FR-66, P14). Watching is read-only and never
// resizes the pane; "Take control" switches to herdr's control stream, where keys, paste
// and window size go to the agent until you hand back. Ctrl+] leaves the terminal.
import { useEffect, useRef, useState } from 'react';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import '@xterm/xterm/css/xterm.css';
import { api } from './api.gen';
import { Chip, useAction } from './ui';

const b64ToBytes = (b: string) => Uint8Array.from(atob(b), (c) => c.charCodeAt(0));
const strToB64 = (s: string) => { const bytes = new TextEncoder().encode(s); let bin = ''; bytes.forEach((x) => { bin += String.fromCharCode(x); }); return btoa(bin); };

const KEYS: [string, string][] = [['Esc', '\x1b'], ['Tab', '\t'], ['Ctrl-C', '\x03'], ['↑', '\x1b[A'], ['↓', '\x1b[B'], ['←', '\x1b[D'], ['→', '\x1b[C'], ['⏎', '\r']];

export function LiveTerminal({ memberId, height = 420, keyBar = false, compact = false }: { memberId: string; height?: number | string; keyBar?: boolean; compact?: boolean }) {
  const box = useRef<HTMLDivElement>(null);
  const termRef = useRef<Terminal | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const [mode, setMode] = useState<'observe' | 'control'>('observe');
  const [status, setStatus] = useState('connecting…');
  const [copied, setCopied] = useState(false);
  const { run } = useAction();

  useEffect(() => {
    const el = box.current;
    if (!el) return;
    const term = new Terminal({ fontSize: compact ? 11 : 12.5, fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace', cursorBlink: mode === 'control', disableStdin: mode !== 'control', convertEol: false,
      theme: { background: '#0e1419', foreground: '#c9d6df' }, scrollback: 2000 });
    const fit = new FitAddon();
    term.loadAddon(fit);
    term.open(el);
    termRef.current = term;
    try { fit.fit(); } catch { /* hidden */ }
    // Ctrl+] leaves the terminal so the page keyboard works again (docs/user/18-keyboard.md).
    term.attachCustomKeyEventHandler((e) => { if (e.type === 'keydown' && e.ctrlKey && e.key === ']') { term.blur(); (el.closest('.live-term') as HTMLElement | null)?.querySelector<HTMLElement>('button')?.focus(); return false; } return true; });
    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
    const url = `${proto}//${location.host}/api/members/${encodeURIComponent(memberId)}/terminal?mode=${mode}&cols=${term.cols}&rows=${term.rows}`;
    const ws = new WebSocket(url);
    wsRef.current = ws;
    const send = (v: any) => { if (ws.readyState === WebSocket.OPEN) ws.send(JSON.stringify(v)); };
    ws.onopen = () => setStatus(mode === 'control' ? 'you have control' : 'watching · read-only');
    ws.onclose = () => setStatus((s) => (s.startsWith('stream ended') || s.startsWith('error') ? s : 'disconnected'));
    ws.onerror = () => setStatus('error: could not connect');
    ws.onmessage = (ev) => {
      const m = JSON.parse(ev.data);
      if (m.t === 'frame') {
        if (mode === 'observe' && m.w && m.h && (m.w !== term.cols || m.h !== term.rows)) term.resize(m.w, m.h); // mirror the pane's size
        term.write(b64ToBytes(m.b), () => send({ t: 'ack', seq: m.seq }));
      } else if (m.t === 'error') setStatus('error: ' + m.message);
      else if (m.t === 'closed') setStatus('stream ended' + (m.reason ? ': ' + m.reason : ''));
    };
    const dataSub = term.onData((d) => { if (mode === 'control') send({ t: 'input', b: strToB64(d) }); });
    const ro = new ResizeObserver(() => {
      if (mode !== 'control') return; // watchers never resize the pane
      try { fit.fit(); send({ t: 'resize', cols: term.cols, rows: term.rows }); } catch { /* hidden */ }
    });
    ro.observe(el);
    if (mode === 'control') term.focus();
    return () => { ro.disconnect(); dataSub.dispose(); ws.close(); term.dispose(); termRef.current = null; };
  }, [memberId, mode, compact]);

  const key = (seq: string) => { const ws = wsRef.current; if (ws?.readyState === WebSocket.OPEN) ws.send(JSON.stringify({ t: 'input', b: strToB64(seq) })); termRef.current?.focus(); };
  const openInHerdr = () => run(async () => {
    const r: any = await api.members.focus(memberId);
    await navigator.clipboard?.writeText(r.command).catch(() => {});
    setCopied(true); setTimeout(() => setCopied(false), 2500);
    return r;
  });
  return <div className="live-term">
    <div className="termbar"><span>● {memberId}</span><Chip c={mode === 'control' ? 'amb' : ''}>{status}</Chip>
      <span className="right" />
      <button type="button" className="lnk small" style={{ color: 'inherit' }} onClick={openInHerdr} title="Focus this pane in herdr and copy the command">{copied ? 'Copied: herdr --session gbi' : 'Open in herdr'}</button>
      {mode === 'observe' ? <button type="button" className="btn p sm" onClick={() => setMode('control')}>⌨ Take control</button>
        : <button type="button" className="btn sm" onClick={() => setMode('observe')}>Hand back</button>}</div>
    <div ref={box} className="xterm-box" style={{ height, background: '#0e1419', padding: 4, borderRadius: '0 0 6px 6px' }} aria-label={`Terminal of ${memberId}${mode === 'control' ? ' (you have control; Ctrl+] to leave)' : ' (read-only)'}`} />
    {keyBar && mode === 'control' && <div className="m-keys">{KEYS.map(([l, s]) => <button type="button" key={l} onClick={() => key(s)}>{l}</button>)}</div>}
    {mode === 'control' && !compact && <div className="small muted" style={{ marginTop: 4 }}>Your keys go to the agent. <kbd>Ctrl</kbd> <kbd>]</kbd> leaves the terminal; Hand back releases control. Taking control is recorded in History.</div>}
  </div>;
}
