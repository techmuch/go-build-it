title: The history
minutes: 5
summary: Everything that happened, who did it and why — the audit trail and the source of insights.
related: history
# The history

Every action, by a person or a persona, is recorded in the **history**: task changes, sessions, questions and answers, reviews, git operations, doctor fixes. It is how you find out *why* something happened.

## Step: Open the history
check: visit /history
show: /history
Open **History** (`g h`). Filter by project, type or person.

## Step: Open an event
check: visit /history/
show: /history
Press **Open ›** on any event. Each type has its own detail view: a session's transcript, a question with its answer, a review with its checks, a git merge with its commits.

> **Pro tip:** *What happened as a result* on an event follows the chain: the question that paused a task, the answer, the session that resumed.

## Step: Ask from the CLI
The same log is searchable from the CLI, which is how your AI harness reads it:

```
$ gbi history list --project practice --limit 5 --json
```

Tick this step off when you have run it.
