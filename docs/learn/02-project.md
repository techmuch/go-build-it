title: Your first project
minutes: 8
summary: A project's purpose, values and roadmap, and how a real repository becomes a project.
related: projects, artifacts
# Your first project

A **project** is a git repository plus what go-build-it knows about it: a purpose, values discovered along the way, requirements and mockups in `docs/`, and a roadmap of milestones that end at review gates.

## Step: Edit the purpose
check: event project.updated
show: /projects/{practice} [data-tour=edit-purpose]
On the practice project's overview, press **Edit** next to *Purpose & values*, change a word and save. Every AI member reads the purpose before it works.

## Step: Add a value
check: event value.*
show: /projects/{practice}/artifacts
Values are principles found while working, like *offline first*. Add one from the command line:

```
$ gbi values add practice --text "Quick to record in the field" --source "lesson 2"
```

> **Pro tip:** values are proposed by the team as they notice them (from your answers, feedback and choices). Accepting one makes every member weigh it from then on.

## Step: Add a milestone with a review gate
check: event milestone.saved
show: /projects/{practice} [data-tour=add-milestone]
Press **+ Milestone** on the roadmap. Give it a name, a scope and a gate: work pauses at the gate until a human reviews it.

## Step: Look at the artifacts
check: visit /projects/{practice}/artifacts
show: /projects/{practice}/artifacts
Open **Artifacts**. It has six sub-tabs: **Requirements, Mockups, Plan, Tracker, Compare and Support files**. The first four each open one HTML **entry point** that you choose. **Support files** lists every file in the repository's `docs/` folder, with its labels.

## Step: Create the standard documents
check: event artifact.init
show: /projects/{practice}/artifacts/requirements [data-tour=docs-init]
The practice project has no entry points yet, so the Requirements tab offers **Create the standard set**. Press it: go-build-it writes the templates (requirements, mockups, plan, tracker), keeps every file that's already there, and records the entry points and labels in `docs/gbi.toml`. The same thing from the CLI is `gbi artifacts init {practice}`.

> **Pro tip:** go-build-it never guesses where a file belongs. Add documents with a label, `gbi artifacts add {practice} docs/notes.md --label research`, and check with `gbi artifacts check {practice}`.

> **Pro tip:** importing your own repository is the same thing: **+ Import project** on Projects, or `gbi projects import --path ~/code/myapp`. go-build-it adds a starter team.
