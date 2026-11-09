section: Getting started
reqs: FR-01, FR-02, FR-02a, FR-02b, FR-03, FR-07, FR-07a, FR-07b, FR-08
# Install and first run

go-build-it is one binary, `gbi`. The quickest way to install it is to give your AI coding agent the repository URL, https://github.com/techmuch/go-build-it, and ask it to install go-build-it. The README has an **Install with an AI agent** section written for the agent to follow. It ends with the doctor reporting that everything works.

To install it yourself:
- macOS and Linux: `curl -fsSL https://raw.githubusercontent.com/techmuch/go-build-it/main/scripts/install.sh | sh`
- Windows: `scripts/install.ps1`

The script checks the download's checksum and signature before installing. Then run:

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

**The doctor** (**Settings → herdr setup**, or `gbi doctor` in a terminal) proves that all of this works. It runs in tiers:
- **static**: herdr is installed, in the supported version range, and has every API method go-build-it needs; your harnesses are installed.
- **live**: it starts the `gbi` session, opens a throwaway pane, runs a command there, and checks that your harnesses are found *inside* the pane.
- **harness** (**Test harnesses**, or `gbi doctor --tier harness`): it starts each harness in herdr, without sending a prompt.
- **prompt** (only when you ask, because it uses a few tokens): it sends one short prompt.

Every problem comes with its evidence and a fix. **Fix** buttons, or `gbi doctor --fix`, apply the automatic ones. The rest say exactly what you need to do. `gbi doctor` works even when the server is not running. See [Troubleshooting](troubleshooting) for every check.

## Learn go-build-it

Open **Learn** in the sidebar (or press <kbd>g</kbd> <kbd>l</kbd>). It has eleven short lessons, from checking your setup to driving gbi from your AI harness, each with Pro tips. They run in a **practice project** whose AI members are scripted, so nothing there uses an AI subscription or touches your repositories. A lesson opens as a coach beside the app: **Show me** points at the control, and steps tick themselves off as you do them. Below the lessons, **How go-build-it thinks** explains why it works the way it does.

To see everything go-build-it is doing from a terminal, run `herdr --session gbi`.

## Windows (beta)

go-build-it supports Windows at the same level as herdr, whose Windows build is in beta (Q9, ADR-009).
- **What CI checks:** go-build-it's Windows CI installs herdr and checks the client, the named-pipe connection, events and the fake member in herdr. Those herdr checks are informational and don't fail the build.
- **Pane commands:** inside herdr panes, go-build-it types PowerShell commands.
- **Missing features:** anything herdr does not yet support on Windows is shown as unavailable rather than worked around.
- **Login service:** `gbi service install` creates a scheduled task that runs at logon.
