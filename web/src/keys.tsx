// Keyboard support shared by the desktop and mobile UIs.
//
// Conventions (documented in docs/user/18-keyboard.md):
// - Everything clickable is reachable with Tab and activates with Enter (and Space for buttons).
// - In a list, table or strip, ↑/↓ (and j/k) move between items; Home/End jump to the ends.
//   In a card grid, ←/→ move too. Enter opens the focused item.
// - In tabs and segmented controls, ←/→ move between options.
// - Dialogs take focus when they open, keep Tab inside, close with Esc, and give focus back.
// - In a text box, ⌘/Ctrl+Enter sends or saves (the panel's primary button).
// - Single-key shortcuts never fire while typing in a text field.
import { useEffect, useRef, type KeyboardEvent as ReactKeyboardEvent, type RefObject } from 'react';

export function isTyping(el: EventTarget | null): boolean {
  if (!(el instanceof HTMLElement)) return false;
  if (el.isContentEditable) return true;
  if (el instanceof HTMLTextAreaElement || el instanceof HTMLSelectElement) return true;
  if (el instanceof HTMLInputElement) return !['button', 'checkbox', 'radio', 'submit', 'reset', 'range', 'color', 'file'].includes(el.type);
  return false;
}

/** Props that make a non-button element (row, card, list item) behave like a button or link. */
export function clickable(onActivate: () => void, opts: { role?: 'button' | 'link'; label?: string } = {}) {
  return {
    onClick: onActivate,
    onKeyDown: (e: ReactKeyboardEvent) => {
      if (e.target !== e.currentTarget) return;
      if (e.key === 'Enter' || (e.key === ' ' && opts.role !== 'link')) { e.preventDefault(); onActivate(); }
    },
    tabIndex: 0,
    role: opts.role || 'button',
    'aria-label': opts.label,
    'data-nav': '',
  } as const;
}

const FOCUSABLE = 'a[href], button:not([disabled]), input:not([disabled]):not([type=hidden]), select:not([disabled]), textarea:not([disabled]), iframe, [tabindex]:not([tabindex="-1"])';

function visible(el: HTMLElement) { return !!(el.offsetWidth || el.offsetHeight || el.getClientRects().length); }
function focusables(root: HTMLElement) { return Array.from(root.querySelectorAll<HTMLElement>(FOCUSABLE)).filter(visible); }

/** Dialog behaviour: initial focus, Tab trap, Esc to close, focus restored on close. */
export function useDialog(ref: RefObject<HTMLElement>, onClose: () => void) {
  const close = useRef(onClose);
  close.current = onClose;
  useEffect(() => {
    const root = ref.current;
    if (!root) return;
    const before = document.activeElement as HTMLElement | null;
    if (!root.contains(document.activeElement)) {
      const first = root.querySelector<HTMLElement>('[autofocus], [data-autofocus]') || focusables(root)[0] || root;
      if (first === root && !root.hasAttribute('tabindex')) root.setAttribute('tabindex', '-1');
      first.focus();
    }
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') { e.preventDefault(); e.stopPropagation(); close.current(); return; }
      if (e.key !== 'Tab') return;
      const els = focusables(root);
      if (!els.length) { e.preventDefault(); return; }
      const first = els[0], last = els[els.length - 1];
      if (e.shiftKey && (document.activeElement === first || !root.contains(document.activeElement))) { e.preventDefault(); last.focus(); }
      else if (!e.shiftKey && document.activeElement === last) { e.preventDefault(); first.focus(); }
    };
    document.addEventListener('keydown', onKey, true);
    return () => { document.removeEventListener('keydown', onKey, true); if (before && document.contains(before)) before.focus(); };
  }, [ref]);
}

function navItems(from: HTMLElement): { items: HTMLElement[]; grid: boolean } {
  const group = from.closest<HTMLElement>('[data-navgroup]') || from.parentElement?.closest<HTMLElement>('tbody, ul, ol, [role=list], [role=listbox]') || from.parentElement!;
  const items = Array.from(group.querySelectorAll<HTMLElement>('[data-nav]')).filter((el) => visible(el) && el.closest('[data-navgroup]') === from.closest('[data-navgroup]'));
  return { items, grid: group.getAttribute('data-navgroup') === 'grid' };
}

/** Move to the item in the next row of a wrapping grid, by position. */
function gridStep(items: HTMLElement[], cur: HTMLElement, dir: 1 | -1): HTMLElement | undefined {
  const r = cur.getBoundingClientRect();
  const cx = r.left + r.width / 2;
  const rows = items.filter((el) => { const b = el.getBoundingClientRect(); return dir > 0 ? b.top > r.bottom - 2 : b.bottom < r.top + 2; });
  if (!rows.length) return undefined;
  const rowTop = dir > 0 ? Math.min(...rows.map((el) => el.getBoundingClientRect().top)) : Math.max(...rows.map((el) => el.getBoundingClientRect().top));
  const row = rows.filter((el) => Math.abs(el.getBoundingClientRect().top - rowTop) < 4);
  return row.reduce((best, el) => { const b = el.getBoundingClientRect(); const d = Math.abs(b.left + b.width / 2 - cx); const bb = best.getBoundingClientRect(); return d < Math.abs(bb.left + bb.width / 2 - cx) ? el : best; }, row[0]);
}

function submitFrom(el: HTMLElement): boolean {
  const scope = el.closest<HTMLElement>('[data-submit-scope], .palette, .viewer-side, .card, .m-card, .modal, form');
  const btn = scope?.querySelector<HTMLButtonElement>('[data-submit]:not([disabled]), .btn.p:not([disabled])');
  if (!btn) return false;
  btn.click();
  return true;
}

/** Install the document-level keyboard conventions once (App and Mobile roots). */
export function useGlobalKeys() {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.defaultPrevented) return;
      const t = e.target as HTMLElement;
      // ⌘/Ctrl+Enter in a text box submits the panel's primary action.
      if (e.key === 'Enter' && (e.metaKey || e.ctrlKey) && isTyping(t)) { if (submitFrom(t)) e.preventDefault(); return; }
      if (e.altKey || e.metaKey || e.ctrlKey || isTyping(t)) return;
      // Tabs and segmented controls: ←/→ move (and select, for segmented controls).
      const bar = t.closest<HTMLElement>('[role=tablist], .seg, nav.tabs, .m-tabs');
      if (bar && (e.key === 'ArrowLeft' || e.key === 'ArrowRight' || e.key === 'Home' || e.key === 'End')) {
        const opts = Array.from(bar.querySelectorAll<HTMLElement>('button, a[href]')).filter(visible);
        const i = opts.indexOf(t);
        if (i < 0) return;
        const j = e.key === 'Home' ? 0 : e.key === 'End' ? opts.length - 1 : Math.max(0, Math.min(opts.length - 1, i + (e.key === 'ArrowRight' ? 1 : -1)));
        e.preventDefault();
        opts[j].focus();
        if (bar.classList.contains('seg')) opts[j].click();
        return;
      }
      // Lists, tables, strips and card grids.
      if (!t.hasAttribute('data-nav')) return;
      const { items, grid } = navItems(t);
      const i = items.indexOf(t);
      if (i < 0) return;
      let next: HTMLElement | undefined;
      const down = e.key === 'ArrowDown' || e.key === 'j', up = e.key === 'ArrowUp' || e.key === 'k';
      if (grid && down) next = gridStep(items, t, 1);
      else if (grid && up) next = gridStep(items, t, -1);
      else if (down || (grid && e.key === 'ArrowRight')) next = items[i + 1];
      else if (up || (grid && e.key === 'ArrowLeft')) next = items[i - 1];
      else if (e.key === 'Home') next = items[0];
      else if (e.key === 'End') next = items[items.length - 1];
      else return;
      e.preventDefault();
      if (next) { next.focus(); next.scrollIntoView({ block: 'nearest' }); }
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, []);
}

/** Global go-to shortcuts: "g" then a letter, "/" for search, "?" for help. Desktop only. */
export const GOTO: [string, string, string][] = [
  ['p', '/', 'Projects'], ['q', '/questions', 'Questions'], ['s', '/sessions', 'Sessions'], ['u', '/queue', 'Queue'],
  ['r', '/reviews', 'Reviews'], ['o', '/options', 'Options'], ['e', '/rehearsals', 'Rehearsals'], ['t', '/archetypes', 'Archetypes'],
  ['a', '/standards', 'Standards'], ['h', '/history', 'History'], [',', '/settings', 'Settings'], ['d', '/docs', 'Docs'],
];

export function useShortcuts(handlers: { search: () => void; help: () => void; go: (to: string) => void }) {
  const h = useRef(handlers);
  h.current = handlers;
  useEffect(() => {
    let pendingG = 0;
    const onKey = (e: KeyboardEvent) => {
      if (e.defaultPrevented || e.altKey || e.metaKey || e.ctrlKey || isTyping(e.target)) return;
      if (document.querySelector('.overlay')) return; // a dialog owns the keyboard
      if (pendingG && Date.now() - pendingG < 1200) {
        pendingG = 0;
        const hit = GOTO.find(([k]) => k === e.key.toLowerCase());
        if (hit) { e.preventDefault(); h.current.go(hit[1]); }
        return;
      }
      pendingG = 0;
      if (e.key === 'g') { pendingG = Date.now(); return; }
      if (e.key === '/') { e.preventDefault(); h.current.search(); return; }
      if (e.key === '?') { e.preventDefault(); h.current.help(); }
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, []);
}
