title: Running work
minutes: 10
summary: The queue, sessions, and watching a member work in herdr.
related: queue, sessions
# Running work

Work is a **queue** of tasks. A task runs as a **session**: a member picks it up in its own herdr pane and worktree, works, and hands the result to review.

## Step: Open the practice queue
check: visit /projects/{practice}/queue
show: /projects/{practice}/queue
Open the practice project's **Queue**. The task *Sighting form* is waiting for a developer.

## Step: Add a task
check: event task.created
show: /projects/{practice}/queue [data-tour=add-task]
Press **+ Task**: title *Species list*, role *developer*. Or:

```
$ gbi tasks add --project practice --title "Species list" --description "Suggest species names (FR-03)." --role developer
```

> **Pro tip:** assign to a **role**, not a person: the dispatcher hands the task to whichever member with that role is free.

## Step: Run a task
check: event session.started
show: /projects/{practice}/queue [data-tour=run-task]
Press **Run** on *Sighting form*. A member starts in herdr and the session appears under **Sessions**.

## Step: Watch it finish
check: event session.completed
show: /sessions
Open **Sessions** (`g s`). The scripted member finishes in seconds; a real one takes longer and shows its context usage as it goes.

## Step: See it in herdr
In a terminal, run `herdr --session gbi`. One workspace per project, each member's worktree grouped under it. Detach with `ctrl+b q`: the members keep running. Tick this step off when you have looked.

> **Pro tip:** **Dispatch** on the queue (or `gbi dispatcher set --auto true`) starts ready tasks on its own, up to the concurrency limit in Settings.
