# go-build-it

go-build-it manages long-running software projects with a team of AI personas and humans. It is one Go binary with an embedded React UI (desktop and mobile), a CLI with the same operations, and an MCP server. Every AI team member runs as a long-running agent harness (Claude Code, Antigravity, Pi, OpenCode) in its own pane of a supervised [herdr](https://herdr.dev) session and its own git worktree. Open `herdr --session gbi` to see everything go-build-it is doing. Teams are per project, made from global persona archetypes (ADR-009, ADR-010).

- Requirements: [docs/requirements.md](docs/requirements.md)
- Delivery plan and tracker: [docs/tracker/](docs/tracker/index.html)
- Decisions: [docs/decisions/](docs/decisions/)
- User docs (also inside the app): [docs/user/](docs/user/)

## Install with an AI agent

Give your AI coding agent (Claude Code, Antigravity, Pi, OpenCode, …) this repository's URL and ask it to install go-build-it. This section is written for that agent. A person can follow it too.

**Rules for the agent:**
- Never touch the person's default herdr session. go-build-it uses its own session, `gbi`.
- Never send a prompt to a harness or start paid work without asking. `gbi doctor --prompt` uses a few tokens, so ask first.
- Never use `sudo` without asking. Nothing below needs it.
- Pass every fix the doctor marks `"kind": "human"` to the person word for word, then wait for them.

**Steps:**

1. **Detect the platform:** `uname -s` and `uname -m` on macOS and Linux, or `$env:PROCESSOR_ARCHITECTURE` on Windows. Windows is beta, like herdr.
2. **Install herdr**, the only runtime dependency, if `herdr --version` fails:
   - macOS: `brew install herdr`
   - Linux: `curl -fsSL https://herdr.dev/install.sh | sh`
   - Windows: `powershell -ExecutionPolicy Bypass -c "irm https://herdr.dev/install.ps1 | iex"`
3. **Download, verify and install gbi.** Run the install script from this repository:
   ```
   curl -fsSL https://raw.githubusercontent.com/techmuch/go-build-it/main/scripts/install.sh | sh
   # Windows: irm https://raw.githubusercontent.com/techmuch/go-build-it/main/scripts/install.ps1 | iex
   ```
   - The script checks the SHA-256 against `SHA256SUMS`, and the Ed25519 signature of `SHA256SUMS` against the release public key `eJ3eKk81oKaKZlNaLjafw1j1GO+Uueyt51jNMCchih4=`. When OpenSSL can't check the signature, the script says so.
   - It installs to `~/.local/bin` (Windows: `%LOCALAPPDATA%\Programs\gbi`), without sudo. `--version` and `--dir` change the defaults.
   - Its last line is `gbi installed: <path> <version>`.
   - Exit codes: `2` means verification failed and nothing was installed (stop and tell the person). `3` means no release is published yet; use step 3b.
   - Make sure the install folder is on `PATH`.
   - **3b. No release yet?** Build from source. This needs Go 1.27+, Node 22+ and git:
     ```
     git clone https://github.com/techmuch/go-build-it && cd go-build-it && make build
     mkdir -p ~/.local/bin && cp bin/gbi ~/.local/bin/gbi
     ```
4. **Find the harnesses.** Check which of `claude`, `agy`, `pi` and `opencode` are installed. If none is, ask the person which one they want and let them install it and sign in. Signing in is a human step.
5. **Run the doctor until it is clean:**
   ```
   gbi doctor --json --fix
   ```
   It needs no running server. Read `exit` and `results`:
   - `0`: done.
   - `1`: every failure has an automatic fix. Run the command again.
   - `2`: something needs a human. Give the person each failing result's `fix.instructions` word for word, wait, then run it again.

   Each result's `id` has a section in [docs/user/20-troubleshooting.md](docs/user/20-troubleshooting.md).
6. **Start it at login, and prove a harness starts:**
   ```
   gbi service install
   gbi doctor --tier harness --json   # starts each harness in herdr; no prompt, no tokens
   ```
   Fix anything it reports, as in step 5.
7. **Open the app:** http://127.0.0.1:7420. Tell the person about **Learn** (`g` `l`). Its lessons run in a practice project that costs nothing. To let you drive gbi later, run `gbi agent guide`. It prints a guide you can save as a skill, for example `gbi agent guide > ~/.claude/skills/gbi/SKILL.md`.

## Build and run

```
make build                  # web UI + binary → bin/gbi
./bin/gbi serve             # http://127.0.0.1:7420, state in ~/.go-build-it
./bin/gbi help              # every UI action has a CLI command
```

## Test

```
make test                                                   # go vet, tsc, Go tests (incl. end-to-end workflow with a scripted agent)
GBI_HOME=$(mktemp -d) PORT=7431 scripts/demo.sh             # seed a demo via the CLI (no AI subscription used)
cd web && GBI_E2E_URL=http://127.0.0.1:7431 npx playwright test   # UAT walkthrough, regenerates docs screenshots
scripts/update_e2e.sh                                       # change log + Update now through a real restart, against a fake signed release
scripts/install_e2e.sh                                      # install.sh against fake signed, tampered and missing releases; then gbi doctor, every tier
scripts/learn_e2e.sh                                        # every `$ gbi` example in the lessons and cookbook, then every lesson completed in the practice project
```

## Release

Every release has an entry in [CHANGELOG.md](CHANGELOG.md), which the app shows when you click its version (FR-06a). Add changes under `## [Unreleased]` as you go, then:

```
gbi dev changelog release 2026.11.10.1      # dates the Unreleased entry; commit it
VERSION=2026.11.10.1 make release           # refuses a version without an entry; cross-compiles, signs SHA256SUMS (ADR-004)
gh release create 2026.11.10.1 release/gbi_* release/SHA256SUMS release/SHA256SUMS.sig --notes-file release/NOTES.md
```
