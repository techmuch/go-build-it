section: Getting started
reqs: FR-01, FR-02, FR-03, FR-07, FR-08
# Install and first run

go-build-it is one binary, `gbi`. Download it for your platform from the latest release, put it on your `PATH`, and run:

```
gbi serve
```

Open http://127.0.0.1:7420. State lives in `~/.go-build-it/` (config, database, worktrees).

To run it in the background at login: `gbi service install`.

## herdr: the agent backend

Every AI team member runs inside [herdr](https://herdr.dev), so install it first:
- macOS: `brew install herdr`
- Linux: `curl -fsSL https://herdr.dev/install.sh | sh`
- Windows: herdr's PowerShell installer. Windows is in beta.

go-build-it starts and supervises its own herdr session, called `gbi`, and keeps it running. Restarting or updating go-build-it does not stop your agents. Your default herdr session is never touched.

**Settings → herdr setup** (or `gbi herdr doctor`) checks each part:
- that herdr is installed and in the supported version range
- that the `gbi` session is running
- that herdr has every API method go-build-it needs
- that herdr's integration for each harness is current; it can update an outdated one for you

To see everything go-build-it is doing from a terminal, run `herdr --session gbi`.
