# ADR-003: Default git workflow (Q5)

Status: Accepted as default · 2026-11-08 · Tracker: P0-05 · The owner may revise; the workflow is a global standard (FR-91), so changes don't need code.

Branch per task `t-<task-id>-<slug>` from the default branch → commits by the developer persona → Courier pushes and opens a PR/MR → Lens assessment → human approval → squash-merge → branch deleted. Releases: Courier tags `YYYY.MM.DD.N` on the default branch, creates the forge release with notes, and publishes Pages. Written as the *Git workflow* standard in `docs/standards/git-workflow.md`.
