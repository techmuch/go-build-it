title: Install and check your setup
minutes: 6
summary: The doctor proves herdr and your harnesses work, and tells you or your AI agent how to fix what doesn't.
related: install, troubleshooting
# Install and check your setup

go-build-it runs every AI team member inside **herdr**, in a session of its own called `gbi`. Before anything else, make sure that chain works: herdr starts, a pane can run a command, and your harness (Claude Code, Antigravity, Pi or OpenCode) starts inside it. The **doctor** checks all of that and explains every problem with evidence and a fix.

> **Pro tip:** installing on a new machine? Give your AI coding agent the GitHub URL and ask it to install go-build-it. The README has a section written for it, ending with `gbi doctor --json --fix`.

## Step: Open the doctor
check: visit /settings/herdr
show: /settings/herdr [data-testid=doctor-tiers]
Open **Settings → herdr setup**. The doctor has already run the **static** and **live** tiers: herdr is installed and supported, the `gbi` session opened a throwaway pane, ran a command and read it back, and your harnesses were found *inside* that pane.

## Step: Start your harnesses
check: event doctor.harness any
show: /settings/herdr [data-testid=doctor-harness]
Press **Test harnesses**. The doctor starts each enabled harness in a throwaway pane and waits until herdr says it is ready. No prompt is sent, so no tokens are used. A harness waiting at a sign-in screen shows that screen as evidence, with a **Needs you** fix.

> **Pro tip:** **Send a test prompt** goes one step further and proves sign-in and quota, but it uses a few tokens, so it only runs when you press it.

## Step: Run it from a terminal
The same checks run without the server, which is what an AI agent uses:

```
$ gbi doctor --tier static --json
```

The exit code says what to do next: `0` all good, `1` run `gbi doctor --fix`, `2` a person must do something. Tick this step off when you have tried it.

> **Pro tip:** `gbi doctor --report gbi-report.md` writes everything (versions, config without secrets, every piece of evidence) into one file to paste into an AI chat or an issue.
