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
Open **Artifacts**: the requirements and the sighting-form mockup come straight from the repository's `docs/` folder. **✎ Edit** opens an editor where you can chat with a persona to change them.

> **Pro tip:** importing your own repository is the same thing: **+ Import project** on Projects, or `gbi projects import --path ~/code/myapp`. go-build-it adds a starter team.
