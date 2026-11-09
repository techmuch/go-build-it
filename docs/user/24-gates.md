section: How go-build-it thinks
reqs: FR-115, FR-116, FR-117, FR-138
# Gates and phases

**In short:** a project's roadmap is a list of milestones, and each ends at a **review gate** (◆) where work waits for a human.

## Why

AI members can produce a lot of work quickly, and quick work can go far in the wrong direction. Gates put a person at the points where direction matters:
- before implementation (a tabletop **dress rehearsal** of the design)
- after it (a UAT rehearsal of the running app)
- at a plain review

Tasks for the next milestone wait for the gate, and the queue shows *waits for gate*.

## How

- A gate has criteria ("a sighting can be saved").
- A **rehearsal** walks the feature step by step, and reviewers add findings to each step. Findings become change requests.
- Only the **owner** approves a gate. Other people can review and add findings, and their names are recorded.

go-build-it's own plan works this way: each phase ends at a gate the owner approves (see the tracker).

Decisions: [ADR-006](https://github.com/techmuch/go-build-it/blob/main/docs/decisions/ADR-006-multi-user.md) (who approves), [ADR-001](https://github.com/techmuch/go-build-it/blob/main/docs/decisions/ADR-001-ui-approach.md) (the roadmap views).
