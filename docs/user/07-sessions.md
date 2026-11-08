section: Using go-build-it
reqs: FR-60, FR-61, FR-62, FR-64, FR-65, FR-66, FR-67, FR-70, FR-71, FR-72, FR-73, FR-74
# Sessions, context and handoffs

A **session** is one headless run of a harness on a task. Each runs in its own git worktree on the task branch.

The agent starts from a prepared context:
- the task
- the requirements in scope
- the project's values and problem definition
- notes from earlier sessions, answers and review feedback

It does not get the whole project history.

![Sessions](screenshots/sessions.png)

**Context usage** is shown live. Claude Code reports it exactly ("measured"); OpenCode, Antigravity and Pi are estimated.

**When context fills,** the harness compacts on its own. This is the default, and each compaction is logged in history.

**Smart handoffs** are experimental (Settings → Experimental). At the threshold, the agent writes a briefing:
- objective and definition of done
- current state
- decisions and why
- dead ends to avoid
- next steps
- pointers to files and artifacts

A fresh session then continues from that briefing. History → *Handoffs vs compaction* shows whether they are ready to become the default (ADR-007).

**Rate limits:** when a subscription limit is hit, the task pauses and resumes automatically when the limit resets.

**Structured and terminal views:** open a session to switch between the two. Claude Code sessions also accept typed input.

![Terminal view](screenshots/session-terminal.png)

**Attachable sessions** are experimental and need tmux (ADR-008). The harness runs interactively, and you take control with `tmux attach -t gbi-S-…`.
