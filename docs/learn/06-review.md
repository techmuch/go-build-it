title: Reviews, rehearsals and options
minutes: 10
summary: How finished work is reviewed, how rehearsals walk a feature, and how option sets let you combine ideas.
related: reviews, rehearsals, options
desktop: only
# Reviews, rehearsals and options

Three ways you steer the work: **reviews** of finished tasks, **dress rehearsals** that walk a feature step by step, and **option sets** that put two or three approaches side by side.

## Step: Review finished work
check: visit /reviews
show: /reviews
A finished task goes to **Reviews**. The reviewer persona (Lens) assesses it first: requirements in scope, standards, the approved mockup against what was built.

## Step: Give a verdict
check: event review.*
show: /reviews
Open the review of *Sighting form* and **Approve** it, or **Request changes** with a comment. Courier merges approved work.

```
$ gbi reviews list --project practice --json
```

## Step: Compare options
check: event options.created
Ask for an option set from the CLI, then open it under **Options** (`g o`):

```
$ gbi options create --project practice --title "Sighting list layout" --options '[{"key":"A","title":"Cards","summary":"Big cards with a photo"},{"key":"B","title":"Compact list","summary":"One line per sighting"}]'
```

> **Pro tip:** tick aspects from *different* options in the combination tray: you can choose "B, but with A's photos".

## Step: Walk a rehearsal
check: event rehearsal.finding
show: /rehearsals
Start a tabletop rehearsal and add a finding to its first step:

```
$ gbi rehearsals create --project practice --title "Record a sighting" --kind tabletop --steps '[{"title":"Open the form","scenario":"Tap New sighting"},{"title":"Save","scenario":"Fill in species and place, then save"}]'
```

Open it under **Rehearsals** (`g e`), select a step and type your feedback.
