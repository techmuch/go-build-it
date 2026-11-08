import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import './styles.css';
import { App } from './App';

try {
  const t = localStorage.getItem('gbi-theme');
  if (t) document.documentElement.setAttribute('data-theme', t);
} catch { /* storage unavailable */ }
if ('serviceWorker' in navigator && location.pathname.startsWith('/m')) navigator.serviceWorker.register('/sw.js').catch(() => {});

createRoot(document.getElementById('root')!).render(<StrictMode><App /></StrictMode>);
