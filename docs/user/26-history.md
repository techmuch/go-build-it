section: How go-build-it thinks
reqs: NFR-10, NFR-11, NFR-12, NFR-14, FR-138
# The history is the audit trail

**In short:** everything that happens is recorded as an event. That includes what people do, what personas do, and what you do through the CLI or your AI harness.

## Why

When a team includes AI members, "why did this happen?" has to have an answer. The history records who did what, when, in which project and **why**:
- task changes and sessions, with transcripts
- questions and answers
- reviews and their checks
- git pushes and merges
- gate decisions
- doctor fixes and updates

Each event type opens a detail view suited to it, and *What happened as a result* follows the chain of events.

## What else uses it

- **Insights:** rework rates, compactions and handoffs are computed from the history. The team can propose improvements to its own prompts and personas from that evidence.
- **Lessons:** Learn ticks steps off by looking for the events your actions produce, so a lesson knows you created a task because the history says so.
- **Your AI harness:** `gbi history list --json` is how it answers "what happened yesterday?".

Decisions: [ADR-006](https://github.com/techmuch/go-build-it/blob/main/docs/decisions/ADR-006-multi-user.md) (attribution), [ADR-012](https://github.com/techmuch/go-build-it/blob/main/docs/decisions/ADR-012-doctor-and-learning.md) (lessons from history).
