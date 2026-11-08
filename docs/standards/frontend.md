# Frontend stack

- React + TypeScript (strict), built with Vite, embedded in the Go binary via `embed.FS`.
- Separate desktop and mobile route trees (`/` and `/m/`).
- No CSS framework: one token-based stylesheet (light and dark) with component classes.
- Data comes only from the HTTP API through the typed client; live updates over SSE.
- Every view has a Playwright e2e test that captures desktop and phone screenshots (FR-112, FR-133).

## Why
One stack means personas reuse patterns across projects, and screenshots double as documentation and review evidence.
