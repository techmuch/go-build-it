# ADR-014: Project documents: HTML entry points, explicit labels, compare and traceability

Status: Accepted · decided by the owner 2026-11-09 · Tracker: P20-01 · Requirements: FR-140–FR-150 · Builds on FR-45, FR-46

## Context

The Artifacts tab listed every `.md`, `.html` and image under `docs/` and grouped them by guessing from the path ("mockups", "decisions"). It didn't know which file was *the* requirements document, couldn't show HTML inside the app, and had no way to compare a mockup with the screen that was built. The owner wants one standard way to handle the program-management side of a project, so that the system works well for people and AI members alike:
- focus on the key documents: requirements, mockups, plan, tracker, and a comparison of mockups with the real app
- the documents are HTML files that may reference PNG or Markdown files
- labels decide where a file belongs, entry points are HTML, and HTML links are how you move between documents
- files are added mostly through the `gbi` CLI, so go-build-it should not guess

## Decision

1. **Six sub-tabs:** Requirements, Mockups, Plan, Tracker, Compare and Support files. The first four each show one **HTML entry point**. Compare is drawn by go-build-it, because it needs decisions and screenshots that no static file has. Support files lists everything under `docs/`.
2. **Explicit only.** Entry points are set with `gbi artifacts entry` and labels with `gbi artifacts add` / `label`, or the same actions in Support files. Adding without a label is an error, and unlabelled files are listed, never guessed. Members are told to use the CLI.
3. **`docs/gbi.toml`** holds the entries and labels. It is versioned with the documents, so a clone carries its structure, and Courier commits each change. A label on a folder (`"docs/mockups/"`) covers its files, which keeps the file short.
4. **HTML is how you navigate.** Several mockup sets are links inside the mockups entry. The viewer shows HTML from `/raw/…` with a `sandbox` Content-Security-Policy, which gives it an opaque origin, so it cannot call go-build-it's API. `/raw` answers with `Access-Control-Allow-Origin: *` so the page can still fetch its own Markdown and JSON. go-build-it adds a small script (`gbi-nav.js`) to the HTML it serves, which tells the viewer where the page is (`postMessage`). The viewer then switches tabs when the page reaches another entry point, and keeps the address bar current. A `.md` link opens rendered.
5. **Markdown inside HTML:** `/gbi-assets/gbi-md.js` defines `<gbi-md src="…">`, which renders Markdown in the page and turns `**FR-nn**` lines and headings into anchors. The requirements and plan templates use it, so prose stays in Markdown, where the editor's accept and reject flow works.
6. **IDs are the links:** `FR-nn` / `NFR-nn`, `data-view` (+ `data-req`, `data-route`) on mockup views, and `reqs` on tasks. Traceability reads them, and nothing else.
7. **Compare is a judgement, not a pixel diff.** Mockups and real data differ in ways that don't matter, so a person accepts the real screen or files a task. `docs/compare.json` stores the decision with the screenshot's SHA-256, so a view comes back only when its screenshot changes. An AI first pass is left for later.
8. **The tracker is a snapshot.** go-build-it's database stays the source of truth. `gbi artifacts snapshot` writes `docs/tracker/tracker.json`, which the tracker HTML reads.

## Consequences

- A project imported without `docs/gbi.toml` shows empty tabs that offer **Choose a file** or `gbi artifacts init`. Nothing appears by itself.
- Hand-written HTML works as long as it uses the IDs. The templates show how.
- The viewer depends on `postMessage` from `gbi-nav.js`. Pages opened outside go-build-it still work as plain HTML, except that `<gbi-md>` needs go-build-it to render.
