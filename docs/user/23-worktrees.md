section: How go-build-it thinks
reqs: FR-80, FR-138
# One worktree per member

**In short:** each AI member works in its own git worktree of the project, on a branch per task. Members never share a working directory.

## Why

Several members working in one checkout would overwrite each other's files, see half-finished changes, and make every test run ambiguous. A git **worktree** is a separate working directory that shares the repository's history, so it is cheap and keeps the work isolated.

## How it flows

1. A task gets a branch, `t-<task-id>-<slug>`, in the member's worktree.
2. The member commits there.
3. **Courier**, the git persona, pushes and opens a pull request.
4. Lens assesses it, and a human approves.
5. The change is squash-merged and the branch deleted.

You can open any worktree yourself. In herdr each member's worktree is grouped under its project's workspace, so `herdr --session gbi` shows who is working where.

> The workflow is a global **standard** (Standards → *Git workflow*), not code. Change it there and every project follows.

Decisions: [ADR-003](https://github.com/techmuch/go-build-it/blob/main/docs/decisions/ADR-003-git-workflow.md), [ADR-009](https://github.com/techmuch/go-build-it/blob/main/docs/decisions/ADR-009-herdr-backend.md).
