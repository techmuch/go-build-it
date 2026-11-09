section: How go-build-it thinks
reqs: FR-10, FR-12, FR-138
# Archetypes and project members

**In short:** roles are defined once, globally, as **persona archetypes**. Each project gets its own **members** made from them, and can tune each one.

## The problem

One global team meant one developer persona for every project. But a finance app and a hiking app need different care: test fixtures for every bank, or one-handed use on a trail. Copying personas per project meant improvements never spread.

## The decision

- **Archetypes are global templates:** role, base instructions, default harness and model, and role standards. They do no work themselves.
- **Members are per project.** A member is an archetype made concrete in one project. It has a name, optional project tuning, and exactly one harness conversation, one git worktree and one herdr pane.
- **Changes flow both ways, on purpose.** When an archetype changes, every member made from it gets a proposal. Members that have not overridden the changed fields can take it in one action. A good project tuning can be **promoted** back to the archetype.

## Why one conversation per member

A member keeps its conversation across tasks, so it remembers the project the way a colleague does. When its context nears the handoff threshold, it rotates to a fresh conversation, starting from a handoff and its notes. The reviewer persona (Lens) is the exception: it starts fresh for every review, so it judges the work rather than remembering how it was made.

Decisions: [ADR-010](https://github.com/techmuch/go-build-it/blob/main/docs/decisions/ADR-010-project-team-members.md), [ADR-007](https://github.com/techmuch/go-build-it/blob/main/docs/decisions/ADR-007-handoff-graduation.md).
