# Upstream watch: herdr and Ovis

Requirements: NFR-30, NFR-31, NFR-32 · Decision: [ADR-009](../decisions/ADR-009-herdr-backend.md)

go-build-it runs every agent in [herdr](https://github.com/herdrdev/herdr). [Ovis](https://github.com/devswha/ovis) is a browser client for herdr, and it's the best reference for how herdr is used in practice. This folder records which revisions of each were last reviewed, what changed, and what go-build-it did about it.

| Project | Last reviewed | Version | Notes |
|---|---|---|---|
| herdr | `ee5bfb5` (2026-10-29) | 0.9.5 | [herdr.md](herdr.md) |
| Ovis | `d763e83` (2026-11-09) | 0.5.1 | [ovis.md](ovis.md) |

## How to review

1. Pull both repositories (`~/projects/herdr`, `~/projects/ovis`).
2. Read the changes since the last reviewed revision:
   - herdr: `git log <rev>..HEAD -- src/app/api src/protocol docs/next CHANGELOG.md`
   - Ovis: `git log <rev>..HEAD -- server/herdr server/conversation.ts server/claude-*.ts server/pty`
3. Diff the herdr API schema: `herdr api schema --json > /tmp/new.json`, then compare it with `internal/herdr/schema/herdr-api.schema.json`. A change there regenerates `internal/herdr/types.gen.go`, and the build fails until the code is updated.
4. Add a dated entry to the project's file: what changed, whether it affects go-build-it, and the follow-up (issue or PR).
5. Update the table above.

The weekly scheduled review (P15-03) follows these same steps and opens an issue or PR.
