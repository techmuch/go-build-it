title: Your team
minutes: 7
summary: Persona archetypes, project members, tuning, and why each member has its own harness.
related: team
# Your team

Each project has its own **team**. AI members are made from global **archetypes** (developer, reviewer, architect, …) and can be **tuned** for the project. Humans join with a role too. Every AI member is one long-running harness conversation in its own herdr pane and git worktree.

## Step: Open the practice team
check: visit /projects/{practice}/team
show: /projects/{practice}/team
Open the practice project's **Team**. These members use the scripted harness; in a real project they use Claude Code or another harness you chose.

## Step: Tune a member
check: event member.tuned
show: /projects/{practice}/team
Open the developer and add a line of project tuning, for example *Keep forms usable with one hand*, then **Save tuning**. Or from the CLI:

```
$ gbi members list practice --json
```

> **Pro tip:** **Diff vs archetype** shows exactly how a member differs from its archetype. When a tuning is good everywhere, **Promote to the archetype** offers it to every project.

## Step: Look at the archetypes
check: visit /archetypes
show: /archetypes
Open **Archetypes** (`g t`). An archetype holds the role, base instructions, default harness and role standards. Changing one proposes the change to every member made from it.

## Step: Add a member
check: event member.added
show: /projects/{practice}/team
Add one more member to the practice team from an archetype, for example QA (Probe). In the practice project new members use the scripted harness automatically.
