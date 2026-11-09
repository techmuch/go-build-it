title: Drive gbi from your AI harness
minutes: 10
summary: Connect Claude Code (or another harness) to gbi through the CLI or MCP, and teach it the workflow.
related: ai-harness, cli
# Drive gbi from your AI harness

Every UI action is a `gbi` command, so the AI harness you already use can drive go-build-it for you: *what's waiting on me?*, *queue this*, *answer Q-12 with A*. See the [cookbook](ai-harness) for many more examples.

## Step: Teach your harness the workflow
`gbi agent guide` prints a guide for AI agents: the commands, how gbi works, and the rules (such as *ask before starting paid work*). Save it as a Claude Code skill:

```
$ gbi agent guide
```

```
mkdir -p ~/.claude/skills/gbi && gbi agent guide > ~/.claude/skills/gbi/SKILL.md
```

Tick this step off when your harness has it.

> **Pro tip:** for other harnesses, put the guide in the project's `AGENTS.md` or the harness's own instructions file.

## Step: Connect over MCP (optional)
Claude Code can also use gbi as an MCP server, with one tool per operation:

```
claude mcp add gbi -- gbi mcp
```

The CLI works in every harness that can run commands; MCP gives Claude Code typed tools. Use either. Tick this step off when you are connected.

## Step: Ask what is waiting
In your harness, ask *"what's waiting on me in gbi?"*. It runs:

```
$ gbi inbox list --json
```

and summarises it. Tick this step off when you have tried it.

## Step: Have it queue work
check: event task.created
Ask your harness: *"add a task to the practice project for the developer: export sightings as CSV"*. It runs `gbi tasks add …`, and the task appears in the practice queue, recorded in the history as done through the CLI.

> **Pro tip:** keep the rule that your harness asks before running tasks: running is what spends your subscription, and gbi's history shows who started what.
