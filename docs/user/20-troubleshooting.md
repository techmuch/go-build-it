section: Reference
reqs: FR-07a, FR-07b, FR-07c, FR-07d, FR-07e
# Troubleshooting with the doctor

The doctor checks that go-build-it can run its agents, and tells you (or your AI agent) what to do when it can't. Run it from **Settings → herdr setup**, or in a terminal:

```
gbi doctor                    # static and live checks (a few seconds, nothing spent)
gbi doctor --harness claude   # also start Claude Code in herdr (no prompt sent)
gbi doctor --tier harness     # start every enabled harness
gbi doctor --prompt           # also send one short prompt to each harness: uses a few tokens
gbi doctor --fix              # apply every automatic fix, then check again
gbi doctor --json             # the same results for an AI agent
gbi doctor --report           # write a Markdown report to paste into an AI chat or an issue
```

`gbi doctor` runs inside the binary, so it works even when `gbi serve` is not running.

## How it works

The checks run in **tiers**. A tier runs only when the tiers before it had no failure:

1. **static**: only looks. Is herdr installed, is its version supported, does it have the API go-build-it needs? Are your harnesses installed, and is each one on the PATH the login service starts go-build-it with?
2. **live**: starts the `gbi` herdr session if it is not running, opens a throwaway `gbi-doctor` workspace in it, runs a command there and reads the output back, and checks that each harness is found *inside the pane*. The workspace is closed afterwards.
3. **harness**: starts each harness in a pane of that workspace and waits until herdr says it is ready. No prompt is sent, so no tokens are used.
4. **prompt** (only when you ask): sends "Reply with the single word READY." and waits for the answer. This proves sign-in and quota end to end.

Your default herdr session is never touched.

Every result has:
- a stable **id** (the headings below)
- a status: ✅ ok, ⚠️ warn, ✖ fail, or – skipped
- the **evidence**: the command, its exit code and the last lines of its output or of the pane
- a **fix**:
  - **Automatic** fixes are applied by `gbi doctor --fix` or the **Fix** button.
  - **Needs you** fixes say exactly what a person must do. An AI agent should pass them on word for word.

**Exit codes for scripts and AI agents:**
- `0`: nothing failed
- `1`: everything that failed has an automatic fix, so run `gbi doctor --fix`
- `2`: something needs a human

> **Pro tip:** when you ask an AI for help, paste the output of `gbi doctor --report`. It has the versions, your configuration (secrets removed) and every piece of evidence.

## Checks

### `herdr.installed`

**What it means:** herdr is not on your PATH. go-build-it runs every agent inside herdr, so nothing works without it.

**Fix:**
- macOS: `brew install herdr`
- Linux: `curl -fsSL https://herdr.dev/install.sh | sh`
- Windows (beta): herdr's PowerShell installer

If you installed herdr somewhere else, set `[herdr] binary = "/full/path/to/herdr"` in `~/.go-build-it/config.toml`.

**Confirm:** `herdr --version`, then `gbi doctor`.

### `herdr.version`

**What it means:**
- **Fail:** herdr is older than the oldest version go-build-it supports.
- **Warning:** herdr is newer than the newest version go-build-it was tested with. It should work, and the weekly upstream review checks new versions.

**Fix:** `herdr update`, or your package manager (`brew upgrade herdr`).

**Confirm:** `gbi doctor`.

### `herdr.api`

**What it means:** this herdr does not offer an API method go-build-it uses. The detail lists the missing methods.

**Fix:** update herdr. If it is already the latest version, go-build-it may need an update too: open the change log (click the version) and check for updates.

**Confirm:** `gbi doctor`.

### `harness.any`

**What it means:** none of the enabled agent harnesses is installed, so no AI team member can run.

**Fix:** install a harness by following its own instructions:
- Claude Code (`claude`)
- Antigravity (`agy`)
- Pi (`pi`)
- OpenCode (`opencode`)

Then sign in to it once in a terminal.

**Confirm:** `gbi doctor --tier harness`.

### `harness.installed`

**What it means:** a harness is enabled in Settings, but its binary is not on your PATH. This is a warning: the other harnesses still work.

**Fix:** install it, or disable it. Disabling is automatic: `gbi doctor --fix` turns it off, and you can turn it back on in **Settings → Harnesses**. If it is installed under another name or folder, set `harnesses.<name>.binary` to its full path in `config.toml`.

**Confirm:** `gbi doctor`.

### `service.installed`

**What it means:** the login service is not installed, so go-build-it and its herdr session will not start when you log in.

**Fix:** `gbi service install`.

**Confirm:** `gbi doctor` shows the service file.

### `harness.on_service_path`

**What it means (warning):** the login service starts go-build-it with a fixed PATH, which was copied from your shell when you ran `gbi service install`. The harness is not on that PATH. This usually means you installed the harness after installing the service. Members may still find it if your login shell adds its folder; the live tier's `harness.on_pane_path` check says for sure.

**Fix (automatic):** run `gbi service install` again from a shell where the harness works. `gbi doctor --fix` does this.

**Confirm:** `gbi doctor`.

### `session.running`

**What it means:** go-build-it could not start or reach its own herdr session, `gbi`.

**Fix:** run `herdr --session gbi server` in a terminal and read what it reports. Common causes are a herdr crash after an upgrade, or a stale socket file after a reboot. Restarting herdr's session clears both: `herdr session stop gbi`, then `gbi doctor`.

**Confirm:** `gbi doctor`.

### `pane.roundtrip`

**What it means:** the session answers, but a command typed into a new pane never printed its output. The shell in the pane may be waiting for input (a prompt in your shell startup files), or failing to start.

**Fix:** run `herdr --session gbi`, open the `gbi-doctor` workspace, and look at the pane. Fix whatever your shell's startup files print or ask, then run the doctor again.

**Confirm:** `gbi doctor`.

### `harness.on_pane_path`

**What it means:** the harness works in your terminal, but not inside herdr panes. Panes get the environment of the herdr server, which can differ from your interactive shell's. This is the most common cause of "the member never starts".

**Fix:**
- **Automatic** (when the doctor knows the full path): it sets `harnesses.<name>.binary` to that path in `config.toml`. `gbi doctor --fix` does this.
- **Or:** add the harness's folder to the PATH your *login* shell sets (for example `~/.zprofile` or `~/.profile`, not only `~/.zshrc`), then restart the session: `herdr session stop gbi`, then `gbi doctor`.

**Confirm:** `gbi doctor`.

### `integration.current`

**What it means:** herdr's integration for this harness is missing or outdated. Members still run without it, but a member's conversation may not be recovered after herdr restarts.

**Fix (automatic):** `gbi doctor --fix`, or **Fix** in Settings → herdr setup.

**Confirm:** `gbi doctor`.

### `harness.starts`

**What it means:** herdr could not get the harness to a ready state:
- **Blocked:** it stopped at a screen that needs you. Usually this is signing in, trusting the folder, or a first-run question. The evidence shows that screen.
- **Did not become ready:** it is slow, crashed, or printed an error. The evidence shows the last lines.

**Fix:** run the harness once in a terminal, in the folder the fix names (`~/.go-build-it/helpers/<harness>`), and finish what it asks. Then run `gbi doctor --harness <name>`.

**Confirm:** `gbi doctor --harness <name>`.

### `harness.answers`

**What it means:** the harness started but did not answer the test prompt. Usually you are not signed in, your plan has no quota left, or the model setting is wrong.

**Fix:** run the harness in a terminal and send it a short message. If that fails too, sign in again, or check your plan and `harnesses.<name>.model` in Settings.

**Confirm:** `gbi doctor --harness <name> --prompt`. This uses a few tokens.
