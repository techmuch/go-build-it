# ADR-016: Users through an admin console, dev mode first, and a personal Assistant for every user

Status: Accepted · decided by the owner 2026-11-20 · Tracker: P27–P33 · Requirements: FR-160–FR-206, NFR-22–NFR-24 · Mockups: [user management](../mockups/user-management.html), Take 1 and section D

## Context

go-build-it is becoming one shared server for a team (decided 2026-11-11). Requirements rev 12 described users, sign-in and per-project access, and the mockups showed three ways to arrange the screens: an admin console (Take 1), access kept with each project's team (Take 2), and stakeholder groups with sign-off policies by stage (Take 3).

Building Last Card with go-build-it (2026-11-15 to 2026-11-19) showed two more things:
- **Multi-user work can't be tried yet.** Every request is the owner. Checking that a reviewer can't approve a gate, or that a second person sees their own inbox, needs sign-in, which is weeks of work.
- **Harnesses were slow and fragile to start.** Each start took seconds and could drop its first prompt; every review started a fresh harness; and Claude Code began stopping new sessions at a usage-credits menu for its default model. A person talking to "the reviewer" or "the architect" in a side chat waited on a cold start each time.

## Decision

1. **Take 1, the admin console.** People & access is a system area with Users, Groups, Roles, Service accounts and AI members; each project has an Access tab with "Check access". It is the shape enterprise admins expect and the easiest to audit. Take 2's roles in the repository (FR-176) are not planned. Take 3's sign-off routes and access review are kept as a later phase (P32), built on Take 1's roles and groups.
2. **Dev mode first.** `gbi serve --dev` adds a bar to every screen that switches the acting user in one click, creates two dev users, records every action as "dev", and explains each allow or deny. It is refused while go-build-it is reachable from other machines. Users, roles, grants and permission checks are built under it before any sign-in, so access rules are tried by clicking, not by reading.
3. **A personal Assistant for every user.** Each user has one AI that works only with them, made from a system-wide Assistant archetype and tuned in their profile. It is always running while they are signed in, its status is on every screen, and every side chat in the UI talks to it. It acts with its user's permissions, and drafts decisions for its user to confirm; it never decides. The project team members keep doing the project's work. The name "Assistant" was chosen over "aide", "partner" and "deputy" as the plainest.
4. **Credentials: Shared or Per user**, as one system setting. A credential is a harness already signed in (the server's, or one made inside a user's own harness folder), or an API key per harness and optionally per model. In Per user mode, work runs on the credentials of whoever started it, and automatic dispatch only starts paid work for a project that names whose credentials it uses.
5. **Warm harnesses.** Assistants of signed-in users and members of projects with queued work are started ahead of need, within a count and memory limit; the rest start on demand. Memory per harness will be measured in dev mode before the default limit is set (Q17).
6. **A harness check before every release** (`scripts/harness_e2e.sh`, NFR-23): a real harness in its own herdr session must work a task without stopping, answer as a helper, resume after a blocking question, and take a second task.

## Consequences

- Multi-user behaviour can be shown and tested weeks before sign-in exists; the sign-in phase (P29) then only adds how people prove who they are.
- Two always-running Assistants and the warm members of active projects use memory on the server even when idle. Idle harnesses use no tokens.
- In Shared mode every user's Assistant spends the server owner's subscription; Settings must say so plainly.
- Side chats change from "pick a persona" to "your Assistant", which asks the right team member. Prompts and tests that name a persona in a side chat change with it.
- Dev mode must never be reachable by others: the refusal when remote access is on, and the warning colour, are part of the requirement, not polish.
