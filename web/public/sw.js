// Minimal service worker so the mobile UI can be installed as a PWA (P6-09).
// It does not cache API responses: live data always comes from the server.
self.addEventListener('install', () => self.skipWaiting());
self.addEventListener('activate', (e) => e.waitUntil(self.clients.claim()));
self.addEventListener('fetch', () => {});
