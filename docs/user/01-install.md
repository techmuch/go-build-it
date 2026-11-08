section: Getting started
reqs: FR-01, FR-02, FR-03
# Install and first run

go-build-it is one binary, `gbi`. Download it for your platform from the latest release, put it on your `PATH`, and run:

```
gbi serve
```

Open http://127.0.0.1:7420. State lives in `~/.go-build-it/` (config, database, worktrees).

To run it in the background at login: `gbi service install`.
