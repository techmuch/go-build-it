# ADR-010: Team members are per project, made from global persona archetypes

Status: Accepted · approved by the owner 2026-11-08 (Q8 and Q10 decided the same day) · Requirements: FR-10–FR-13, FR-12a–FR-12e, FR-69, FR-76, FR-81 · Tracker: P12

## Context

Until rev 3, AI personas and teams were global, and a team could be assigned to several projects. With herdr as the backend ([ADR-009](ADR-009-herdr-backend.md)), each AI team member becomes a long-running agent with its own harness conversation, worktree and herdr pane. Those things only make sense inside one project:

- A conversation is tied to the folder it was started in.
- A worktree belongs to one repository.
- A pane lives in one project's workspace.

## Decision

1. **Persona archetypes are global.**
   - An archetype holds a role, base instructions, a default harness and model, and role-specific standards. It does no work itself.
   - go-build-it ships one starter archetype per built-in role.
2. **Team members are per project.**
   - Adding a member to a project instantiates an archetype.
   - The project can tune the member: extra or replacement instructions, harness, model, display name and project standards.
   - The member keeps a link to its archetype, and the UI shows where it differs.
3. **Binding.** Each AI team member has exactly one:
   - harness and model
   - current conversation ID
   - long-lived worktree (`worktrees/<project>/<member>`, with a new branch per task)
   - herdr pane

   One member does one task at a time. Parallel work means adding members, such as a second developer.
4. **Archetype changes are proposals.**
   - When an archetype changes, go-build-it lists the affected members. The user applies the change per member, and that member's tuning is kept.
   - Tuning made in a project can be promoted back to the archetype through the artifact editor.
   - **Q10 (decided):** the review also offers one action that applies the change to every member that has not overridden the changed fields.
5. **Humans stay global.** A human user is the same identity everywhere and holds a role per project.
6. **Memory lives in go-build-it.**
   - The persona, the project's values and decisions, and notes from the member's finished tasks are stored by go-build-it.
   - The conversation is a working cache. **Q8 (decided):** a member keeps one conversation across tasks until context reaches the handoff threshold (75%, `experimental.handoff_threshold`). It is then rotated to a new conversation ID that starts from a handoff plus that stored memory.
7. **Reviewers stay independent.**
   - Lens (the Reviewer role) starts a fresh conversation for each review, so it doesn't review work it argued for.
   - Builders keep long-running conversations.

## Consequences

- **Schema.** The `teams`, team-assignment and global persona tables are replaced. The new tables are `archetypes` and `project_members`, with AI fields: archetype, tuning, harness, model, conversation, worktree and pane. No members or teams exist in any install yet, so no data migration is needed beyond the schema.
- **Interfaces.** The Team view (A-v10) splits into an Archetypes library and a per-project Team tab. Task assignment targets a role or a specific member. The CLI, API and MCP follow the route table.
- **Instruction files.** The generated instruction file (`CLAUDE.md` / `AGENTS.md`) is written per member into that member's worktree, combining global standards, project overrides, archetype instructions and member tuning.
- **Catalog cards.** A project's card shows its team members and their herdr state, instead of an assigned global team.
