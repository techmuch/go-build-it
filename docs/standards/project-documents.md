# Project documents

Every project keeps the same documents, so people, AI members and go-build-it find them without guessing (FR-140–FR-150, ADR-014).

## Entry points

Four HTML files under `docs/`, one per kind. Each is chosen explicitly and recorded in `docs/gbi.toml`:

| Role | Default file | What it holds |
|---|---|---|
| requirements | `docs/requirements.html` | renders `docs/requirements.md` with `<gbi-md>` |
| mockups | `docs/mockups/index.html` | links to each set (web, phone, …) |
| plan | `docs/plan.html` | renders `docs/plan.md` |
| tracker | `docs/tracker/index.html` | reads `tracker.json`, a snapshot of go-build-it's tasks |

- Start a project with `gbi artifacts init <project>`. It writes the templates and never overwrites a file.
- Change an entry point with `gbi artifacts entry <project> <role> docs/<file>.html`. It must be HTML.
- Several sets of one kind go *inside* the entry HTML, as links. HTML links are how you and the viewer move between documents.

## Labels

Every file under `docs/` has at least one label from this list: `requirements`, `mockup`, `plan`, `tracker`, `screenshot`, `decision`, `research`, `asset`, `support`.

- **Add files with the CLI, with a label:** `gbi artifacts add <project> docs/<file> --label <label>`. A label is required.
- **A folder label covers the folder's files:** `gbi artifacts label <project> docs/mockups/ --label mockup`.
- **Nothing is guessed.** A file added any other way is *unlabelled* until someone labels it. A task isn't done while it leaves unlabelled files behind.

## IDs

- **Requirements:** `**FR-nn**:` (or `NFR-nn`) at the start of a line. Never reuse or renumber an ID.
- **Mockup views:** an element with `data-view="v01"`, plus `data-req="FR-01 FR-02"` for the requirements it covers and `data-route="/path"` for the screen it shows. Give it the same `id` so `#v01` links work.
- **Tasks:** list their requirements in `reqs` and name their views (`v01`) in the title or description.
- **Screenshots:** of the built app, one per view, named after the view (`docs/screenshots/v01.png`) and labelled `screenshot`. UI tests should write them.

## Checking

- `gbi artifacts check <project>` lists missing entry points, unlabelled files, broken links and unknown IDs. Keep it clean.
- `gbi trace list <project> --gaps` lists requirements without tasks, and views that drifted or were never reviewed.
- **Compare** (`gbi compare list|accept|task`) is how a person decides whether a built screen matches its mockup.
