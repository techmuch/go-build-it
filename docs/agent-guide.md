---
name: gbi
description: Drive go-build-it (gbi) from the command line. Use it when the user asks about their go-build-it projects, tasks, questions, reviews, team members or sessions, or wants work queued, run, answered or approved in gbi.
---

# Driving go-build-it (gbi)

go-build-it manages software projects with a team of AI members and humans. Every action in its web UI is also a `gbi` command (and an MCP tool, through `gbi mcp`). You are working **for the person**: act on their requests, and bring them decisions that are theirs to make.

## Rules

1. **Ask before starting paid work.** `gbi tasks run`, `gbi dispatcher tick`, `gbi dispatcher set --auto true` and `gbi doctor --prompt` start AI harnesses that use the person's subscription. Say what you are about to start and get a yes first. Reading anything is always fine.
2. **Decisions are the person's.** Do not answer questions, approve reviews or gates, or choose options on your own judgment. Show the question or review, give your recommendation, and act once they decide (for example "answer Q-12 with A").
3. **Never touch the person's default herdr session.** gbi runs every member in its own herdr session, `gbi`. Never stop or kill panes in it.
4. **Use `--json`** when you need to read results, and quote IDs (T-12, Q-3, R-1, S-4) back to the person.
5. **Say what you did.** After a change, name the command and the result, so it matches what they see in gbi's history.
6. **Add documents with a label.** A file you put under a project's `docs/` must be added with `gbi artifacts add PROJECT docs/… --label LABEL`. gbi never guesses labels. Follow the **Project documents** standard (`gbi standards get project-documents`).

## Where things are

```
gbi inbox list --json                  # everything waiting on a human: questions, reviews, gates, option sets
gbi projects list --json               # projects with stage, team and status
gbi tasks list --project ID --json     # the queue (status: queued, running, review, done, failed, paused)
gbi sessions list --live --json        # members working right now; "blocked" means a member needs the person
gbi history list --project ID --limit 20 --json   # what happened, newest first
gbi questions list --json              # open questions, blocking first
gbi reviews list --json                # work waiting for review
gbi members list PROJECT --json        # the project's team
gbi help                               # every command; gbi help GROUP COMMAND for flags
```

## Common requests

| The person says | You run |
|---|---|
| "What's waiting on me?" | `gbi inbox list --json`, then summarise it, blocking items first |
| "Add a task for the developer" | `gbi tasks add --project ID --title "…" --description "…" --role developer` |
| "Start it" (after they said yes) | `gbi tasks run T-…` |
| "Answer Q-12 with A" | `gbi questions answer Q-12 --choice A` (add `--answer "…"` for free text) |
| "Approve R-4" | `gbi reviews verdict R-4 --verdict approve --comment "…"` |
| "Who is blocked?" | `gbi sessions list --live --json`: the blocked ones need the person in the UI (Sessions → Open terminal) |
| "What happened yesterday?" | `gbi history list --since 2026-11-08T00:00:00Z --json` |
| "Tune Forge to always add a fixture" | `gbi members tune forge@ID --tuning "- Always add a fixture."` |
| "What's missing in the docs?" | `gbi artifacts check ID --json` and `gbi trace list ID --gaps --json` |
| "Which screens don't match their mockups?" | `gbi compare list ID --json`: `drifted` and `not-reviewed` need the person (accept or file a task is their call) |
| "Add this mockup to the project" | write it under `docs/mockups/`, mark views `data-view="v05" data-req="FR-…"`, then `gbi artifacts add ID docs/mockups/… --label mockup` |
| "Update the tracker" | `gbi artifacts snapshot ID` |
| "Is gbi working?" | `gbi doctor --json`; give the person every fix whose `kind` is `human`, word for word |

## How gbi works (so your answers are right)

- **Projects and their teams.** A project is a git repository with requirements, values, a roadmap of milestones with review gates, and a queue. Each project has its own team: AI members made from global persona archetypes (developer, reviewer, …) plus humans.
- **Where members run.** Every AI member runs as a long-running harness (Claude Code, Antigravity, Pi, OpenCode) in its own herdr pane and git worktree. `herdr --session gbi` shows them all.
- **Sessions.** Tasks run as sessions. A finished task goes to review (the Lens persona assesses it, then a human approves), and Courier merges it.
- **Questions.** Members ask instead of guessing. A blocking question pauses its task until the person answers.
- **Documents.** Each project has HTML entry points for requirements, mockups, plan and tracker (`docs/gbi.toml`), labels on every file under `docs/`, and stable IDs: `FR-nn` requirements, `data-view` mockup views, tasks with `reqs`. Compare pairs each view with its screenshot (`docs/screenshots/v01.png`).
- **History.** Everything is recorded in the history, including what you do through these commands.
- **Learning.** The person can learn gbi in **Learn** (`g l` in the UI). Its lessons run in a practice project (`practice`) that uses a scripted harness and costs nothing. Suggest it when they are new.
