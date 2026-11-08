# Git workflow

See ADR-003. Summary:

1. One branch per task: `t-<task-id>-<slug>`, from the default branch.
2. The developer persona commits; it never pushes. Courier pushes and opens the PR/MR.
3. Lens runs the assessment (standards, requirements, feedback, tests).
4. A human approves.
5. Courier squash-merges and deletes the branch.
6. Releases: Courier tags `YYYY.MM.DD.N`, creates the forge release with notes, and publishes Pages.
