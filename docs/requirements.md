# go-build-it — Requirements

Status: Draft · Last updated: 2026-11-20 (rev. 15: nothing to set up on your own machine: a **setup wizard** in the browser sets up each person's Assistant, signing the harness in through a web terminal, with an API key as the fallback; it can be skipped, and what is left stays visible and easy to find; a shared server starts in multi-user mode with Per user credentials; the CLI can point at a server; FR-219–FR-228; rev. 14: on a multi-user Linux server, every user gets an isolated **workspace** (a container or microVM) for their Assistant, the agent work they start, their worktrees and credentials, without the Docker Engine, FR-207–FR-218, NFR-20 and NFR-25; rev. 13: user management follows Take 1, the admin console (Q13); a **dev mode** with a user switcher for trying multi-user work on one machine; a **personal Assistant** for every user, always running while they are signed in and the one you talk to in every side chat; shared or per-user **harness credentials**; **warm harnesses** started ahead of need; a **harness check** before every release; evidence gaps need a reason to approve, FR-190–FR-206, FR-112a, NFR-23–NFR-24. rev. 12: users, sign-in and per-project access control for one shared server: local accounts first and OIDC next, a loopback-trust setting, statuses, groups, custom roles, AI members under the same rules, sign-off routing and access reviews, FR-160–FR-189; answers Q1 and Q4; three takes are in the [user management mockups](mockups/user-management.html). Rev. 11: a phone that opens a desktop address gets the matching mobile screen, with a per-device Desktop site choice, and the desktop UI keeps its navigation behind ☰ on narrow screens, FR-35a; the remote access QR code opens the mobile UI, FR-20c. Rev. 10: each harness reports its models, and archetypes and members choose one from a searchable list, FR-63a; the logo and title lead to the landing page, FR-34a. Rev. 9: starter archetypes with full role instructions that upgrade when unedited, FR-12f; new installations start with the starter standards, FR-50a; a remote access checklist that proves go-build-it is served, private and not clashing with other services, with a QR code, FR-20a–FR-20d, see [ADR-015](decisions/ADR-015-remote-access.md). Rev. 8: docs/gbi.toml controls the Artifacts sub-tabs, which ones exist, their order and their names, including your own roles such as vision, FR-151. Rev. 7: project documents with HTML entry points, explicit labels, a viewer that follows links, mockup-to-app comparison and traceability, FR-140–FR-150, see [ADR-014](decisions/ADR-014-project-documents.md). Rev. 6: go-build-it answers a harness's folder trust prompt for folders it created, FR-07f, see [ADR-013](decisions/ADR-013-folder-trust.md). Rev. 5: a doctor that proves gbi can start herdr and a harness and explains failures to people and AI agents, FR-07a–FR-07e; install by giving an AI agent the GitHub URL, FR-02a–FR-02b; lessons, a practice project, harness control and design rationale, FR-134–FR-139; see [ADR-012](decisions/ADR-012-doctor-and-learning.md). Rev. 4: in-app change log and guided updates, FR-06a–FR-06e, see [ADR-011](decisions/ADR-011-changelog-guided-updates.md). Rev. 3: herdr as the only agent backend; per-project team members from global archetypes, see [ADR-009](decisions/ADR-009-herdr-backend.md) and [ADR-010](decisions/ADR-010-project-team-members.md))

## 1. Overview

go-build-it is a Go application with an embedded React frontend that manages long-running software project development. It is installed globally on a machine and provides a web UI and a CLI. A team of AI personas and humans works through it to take projects from problem definition to released software. The app coordinates external agent harnesses such as Claude Code, Antigravity and OpenCode. Every harness runs interactively inside [herdr](https://herdr.dev), a terminal workspace manager for coding agents, in a herdr session that go-build-it owns. Opening that session in any terminal shows everything go-build-it is doing, organized by project and team member.

### Conventions

- **SHALL**: required for the release the requirement is assigned to.
- **SHOULD**: strongly desired. It can be deferred, but the reason must be recorded.
- **MAY**: optional.
- Requirement IDs (`FR-xx`, `NFR-xx`) are stable. Tests, the UAT walkthrough and the docs refer to them by ID.

## 2. Glossary

| Term | Definition |
|---|---|
| **Project** | A software product managed by go-build-it. It is backed by a git repository and has its own requirements, values, roadmap, task queue and history. |
| **Team** | The team members working on one project. Each project has its own team. |
| **Team member** | Either an **AI team member** or a **human user**, holding one role in a project. |
| **Persona archetype** | A global, reusable template for an AI team member: role, base instructions, default harness and model, and role-specific standards. Archetypes do no work themselves. |
| **AI team member** | An archetype made concrete inside one project, with optional project-specific tuning. It is bound to one harness and model, one long-running harness conversation, one git worktree and one herdr pane. Also called an *AI persona*. |
| **Role** | A responsibility within the team. See §4.3. |
| **Agent harness** | An external agent CLI (e.g. Claude Code, Antigravity, OpenCode, Pi-agent) that go-build-it runs interactively in a herdr pane to carry out tasks. |
| **herdr** | The terminal workspace manager that owns every harness process for go-build-it. It reports agent state (working, blocked, done, idle) and exposes a socket API. It is the only agent backend. |
| **gbi herdr session** | The named herdr session (`gbi`) that go-build-it starts and supervises. Open it with `herdr --session gbi`. |
| **Conversation** | A harness's own resumable conversation (e.g. a Claude Code session ID). Each AI team member keeps one current conversation across tasks until it is rotated. |
| **Session** | The span of one AI team member working on one task: from the prompt that starts the task to its completion, inside the member's conversation. |
| **Handoff** | A curated briefing produced when a session ends, mainly when its context window is nearly full. The next session starts from it instead of from a raw memory dump. Smart handoffs are an **experimental** feature; the MVP relies on the harness's own compaction (§8.2). |
| **Artifact editor** | The view for changing a project artifact, standard or persona definition: chat with an AI agent to make changes, or edit directly in a rich editor. Saving creates a new revision. |
| **Global standards** | Rules and conventions that apply to every project, e.g. the standard backend stack, frontend stack and test requirements. |
| **Values** | A project's guiding principles, discovered as work proceeds. They are used to judge and generate solutions. |
| **Question** | A request for input sent from a team member (usually an AI persona) to a human. It carries the context and artifacts needed to answer it. |
| **Review gate** | A planned point in a project's roadmap where work pauses for human review, such as a dress rehearsal. |
| **Dress rehearsal** | A guided, feature-by-feature walkthrough of the application, done as a tabletop exercise before implementation or as a UAT run after it. The human reviewer gives feedback on each feature. |
| **Option set** | Two or three alternative approaches to a problem, presented for the human to choose among or combine. |

## 3. Platform, installation and runtime

- **FR-01**: The application SHALL be a single Go binary with the React frontend embedded. Its one required runtime dependency is herdr (FR-07).
- **FR-07**: go-build-it SHALL require herdr and SHALL use it as its only agent backend:
  - It SHALL check that herdr is installed and within the supported version range, and SHALL guide the user to install or update it (e.g. `brew install herdr`).
  - It SHALL start and supervise a headless herdr server for a named session `gbi` (`herdr server --session gbi`), and restart it if it stops.
  - It SHALL check that herdr's integration for each enabled harness is installed and current (`herdr integration status`), and SHALL offer to install or update it.
  - It SHALL check herdr's advertised API methods at connect time and report a clear error when a required method is missing.
- **FR-07a**: go-build-it SHALL provide a **doctor** (`gbi doctor`, and **Settings → herdr setup** in the UI) that runs inside the `gbi` binary itself. It SHALL work when `gbi serve` is not running, because a broken server is one of the things it diagnoses.
- **FR-07b**: The doctor SHALL run its checks in tiers. Each tier runs only if the tiers before it passed:
  1. **static**: the herdr checks of FR-07, the login service, and each enabled harness's binary found on the PATH the login service uses (FR-02).
  2. **live**: go-build-it starts the gbi herdr session if it is not running, opens a throwaway `doctor` tab in it, runs a command there and reads its output back, checks that each enabled harness's binary is found *inside the pane*, then closes the tab.
  3. **harness**: for each enabled harness (or the one named), the doctor starts the harness in a throwaway pane and waits until herdr reports it ready. A harness that stops at a sign-in, trust or first-run screen, exits, or does not start within the time limit fails the check, and the doctor shows the last lines of the screen. No prompt is sent, so this uses no tokens.
  4. **prompt**: the doctor sends one short prompt and waits for the reply, which proves sign-in and quota end to end. It uses a few tokens, so it SHALL run only when asked for (`--prompt`, or a button labeled with that cost). It SHALL never run automatically.

  The static and live tiers run by default. Throwaway tabs and panes SHALL be closed afterwards, and the doctor SHALL never touch the user's default herdr session.
- **FR-07c**: Every doctor result SHALL serve both a person and an AI agent. It SHALL carry:
  - a stable `id` (e.g. `harness.not_on_pane_path`)
  - a status: ok, warn, fail or skipped
  - what was checked, in plain words
  - **evidence**: the command run, its exit code, and the last lines of its output or of the pane
  - a **fix**, which is one of two kinds. An *automatic* fix is one the doctor can apply itself (`gbi doctor --fix`). A *human* fix (signing in to a harness, sudo, approving a package install) says exactly what the person must do, written so an AI agent can pass it on word for word.
  - a link to the troubleshooting section of the user docs for that `id`
- **FR-07d**: `gbi doctor --json` SHALL print the results as JSON. It SHALL exit 0 when nothing failed, 1 when every failure has an automatic fix, and 2 when a failure needs a human. `gbi doctor --fix` SHALL apply the automatic fixes and run the checks again. `gbi doctor --report` SHALL write one Markdown file to paste into an AI chat or an issue. The file holds the versions, the platform, the configuration with secrets removed, and every result with its evidence.
- **FR-07e**: The user docs SHALL have a troubleshooting page with one section per doctor `id`: what it means, how to fix it, and how to confirm the fix.
- **FR-07f**: When a harness stops at its **folder trust** prompt ("Do you trust this folder?") in a folder inside go-build-it's own directory (member worktrees, helpers, the practice project), go-build-it SHALL answer yes by pressing the keys a person would, so starting an agent needs no human. This SHALL be on by default and can be turned off (`[agents] trust_folders`, **Settings → Agent harnesses**). Each answer SHALL be recorded in the history (`agent.trusted`). Folders outside go-build-it's directory, and every other prompt (sign-in, permissions), SHALL still be left to the person. The doctor's harness tier SHALL do the same, and when the setting is off SHALL name the trust prompt in its human fix. See [ADR-013](decisions/ADR-013-folder-trust.md).
- **FR-08**: The background service (FR-02) SHALL ensure both go-build-it and its herdr session are running at login. Stopping or updating go-build-it SHALL NOT stop running harnesses, because herdr owns them.
- **FR-02**: The application SHALL be installable globally on a machine and SHOULD run as a background service (launchd on macOS, systemd on Linux, a Windows service on Windows).
- **FR-02a**: An AI coding agent SHALL be able to install go-build-it given only the GitHub repository URL. The README SHALL contain an **Install with an AI agent** section that the agent can follow step by step:
  1. detect the platform
  2. install herdr
  3. download the release for the platform and verify its checksum and signature against the public key printed in the README
  4. put `gbi` on the PATH without sudo
  5. find which harnesses are installed, and ask the person which to use when none is
  6. run `gbi doctor --json --fix` until nothing is left to fix, passing every human fix to the person word for word
  7. install the login service, run the harness tier, and open the UI

  The section SHALL also state the rules: never touch the person's default herdr session, never send a prompt or start paid work without asking, never use sudo without asking. While no release is published, the section SHALL say so and give the build-from-source steps instead.
- **FR-02b**: The repository SHALL include `scripts/install.sh` (macOS and Linux) and `scripts/install.ps1` (Windows) that perform steps 3–4 of FR-02a in one command. Each SHALL accept a version and an install directory, and SHALL refuse to install a download whose checksum does not match. It SHALL also refuse one whose signature fails when it can verify the signature, and say so when it cannot. An `AGENTS.md` at the repository root SHALL point AI agents to the README section.
- **NFR-01**: macOS SHALL be the initial target platform. Linux and Windows SHALL be supported later. Code SHALL NOT assume macOS-only paths, shells or tools without a cross-platform abstraction. Platform support SHALL follow herdr's: a platform where herdr is in beta (currently Windows) is supported at the same level.
- **FR-03**: All application state SHALL live under `~/.go-build-it/` (`%USERPROFILE%\.go-build-it\` on Windows). This includes:
  - configuration files
  - the SQLite database
  - logs and session transcripts

  herdr's own state lives in herdr's config directory under its `gbi` session.
- **FR-04**: Persistence SHALL use SQLite.
- **FR-05**: The application SHALL use calendar versioning in the format `YYYY.MM.DD.N`, where `N` is the release number for that day, starting at 1. The application is not intended to be consumed as a Go module.
- **FR-06**: The application SHALL be able to install new releases automatically:
  - Releases SHALL be verified by checksum and SHOULD be signed.
  - Because herdr owns the harness processes (FR-08), an update MAY be applied while AI team members are working; go-build-it SHALL reconnect to the gbi herdr session after restarting. Updating herdr itself is left to herdr's own updater or package manager, and go-build-it SHALL warn when herdr falls outside the supported range.
  - The previous version SHALL be kept so the user can roll back.
  - The user SHALL be able to turn automatic updates off.
- **FR-06a**: go-build-it SHALL keep a change log of its own releases in `CHANGELOG.md` at the root of its repository. The change log has one entry per CalVer version (FR-05) with its release date and its changes grouped as Added, Changed, Fixed and Removed, plus an Unreleased entry for work not yet released. The change log SHALL be embedded in the binary. Each GitHub release's notes SHALL come from its change log entry, and a release SHALL NOT be built without one.
- **FR-06b**: Clicking the version number SHALL open the change log, in the desktop sidebar and on the mobile More screen. The change log SHALL list entries newest first and mark the installed version. Development builds show the Unreleased entry as the installed one.
- **FR-06c**: The change log SHALL offer **Check for updates**. The result SHALL show the installed version, the latest release, and when the check ran. When the latest release is newer, the result SHALL also list the change log entries of every release between the two, so the user can see what the update brings. When no release can be found, the result SHALL say why (e.g. offline, or no release published yet).
- **FR-06d**: When a newer release is available, the change log SHALL offer **Update now**. The update SHALL run as named steps, and each step SHALL be shown as pending, running, done (with how long it took) or failed (with the reason). The steps are:
  1. find the release
  2. download it, with progress
  3. verify the signature
  4. verify the checksum
  5. test that the new binary runs and reports the expected version
  6. install it, keeping the current version for rollback
  7. restart go-build-it
  8. reconnect to the gbi herdr session and the members' running work

  Progress SHALL survive the restart: the restarted go-build-it SHALL complete the remaining steps, and the change log SHALL show them finishing. A step that fails before the install step SHALL leave the running version untouched and offer **Try again**. A failure after it SHALL offer **Roll back**. The CLI SHALL offer the same (FR-30).
- **FR-06e**: Updates installed by the automatic updater (FR-06) SHALL run the same steps and SHALL be visible in the change log in the same way. Every update, whether it succeeds, fails or is rolled back, SHALL be recorded in the history (NFR-10).

## 4. Users, team and roles

### 4.1 Team members

- **FR-10**: A team member SHALL be either an AI team member or a human user.
- **FR-11**: Each project SHALL have its own team. The user SHALL be able to add, tune and remove team members per project. There are no global teams.
- **FR-12**: The application SHALL maintain global **persona archetypes**. Each archetype SHALL be configurable with: role, base instructions, default harness, default model, and any role-specific standards that add to the global standards. go-build-it SHALL ship a starter archetype for each built-in role (§4.3).
- **FR-12a**: An AI team member SHALL be created in a project from an archetype. The project MAY tune it: extra or replacement instructions, a different harness or model, a display name, and project-specific standards. The member SHALL keep a link to its archetype and SHALL show where it differs.
- **FR-12b**: When an archetype changes, the application SHALL show which project members it affects and SHALL let the user apply the change to each of them, keeping their project tuning. Changes SHALL NOT be pushed to members silently.
- **FR-12c**: Tuning made in a project MAY be promoted back to the archetype, through the artifact editor's normal accept or reject flow.
- **FR-12d**: Each AI team member SHALL be bound to exactly one harness and model, one current harness conversation, one long-lived git worktree, and one herdr pane in the gbi herdr session (§8.1). One member works on one task at a time; more parallel work means more members.
- **FR-12e**: Human users SHALL remain global identities (§4.2). A human SHALL hold a role per project.
- **FR-12f**: *(rev 9)* Each starter archetype SHALL carry complete role instructions: its mission, what good work looks like, how it works (naming the gbi commands it uses), when it hands back, and what it never does. They SHALL hold only what is specific to the role, because the standards, the task and the project reach the agent separately (FR-53). They draw on established practice: small vertical slices tied to requirement IDs (PM); quality-attribute scenarios, option sets and ADRs (architect); test-first, small diffs, never weakening a test (developer); risk-based and exploratory testing (QA); approving changes that improve the system, with evidence (reviewer); small revertible batches and release notes written for users (release manager). When go-build-it ships improved starter instructions, an archetype whose instructions were never edited SHALL get them as a new revision, keeping the owner's harness, model, name and standards; members SHALL move to it through the change review (FR-12b). The agent that writes release notes SHALL follow the release manager's instructions.
- **FR-13**: Archetypes and project members' tuning SHALL be edited in the artifact editor (FR-45a). The user can write the definition directly, or chat with an AI agent that helps develop it (e.g. interviews the user, suggests instructions, checks for conflicts with global standards).

### 4.2 Access and identity

In **solo mode** (one owner, the default for a new installation), authentication is delegated to the machine and to Tailscale, as below. In **multi-user mode** (§4.4, rev 12), everyone signs in and FR-16 and FR-17 change as noted.

- **FR-15**: The application SHALL detect how each request reaches it and classify it as one of:
  1. **Local**: a loopback connection (`localhost` / `127.0.0.1` / `::1`).
  2. **Tailscale Serve**: proxied by `tailscale serve` on the tailnet.
  3. **External interface**: a direct connection on a non-loopback network interface.
- **FR-16**: **Local** requests SHALL NOT be challenged. They SHALL be attributed to a configurable default local user. *(rev 12)* In multi-user mode this applies only when loopback trust is on (FR-164).
- **FR-17**: **Tailscale Serve** requests SHALL be treated as trusted, the same as local requests, because the user has already authenticated with Tailscale. When Tailscale identity headers are present (e.g. `Tailscale-User-Login`), the request SHOULD be attributed to that Tailscale user, so actions in the history show who performed them. *(rev 12)* In multi-user mode a Tailscale identity only finds its linked user (FR-165); it SHALL NOT grant access on its own.
- **FR-18**: **External interface** requests SHALL be challenged before access is granted. *(rev 12)* The challenge is the sign-in of §4.4 (Q1 answered).
- **FR-19**: By default the application SHALL listen only on loopback. Listening on external interfaces SHALL require explicit configuration.
- **FR-20**: The application SHALL help the user set up Tailscale Serve, for example by detecting whether Tailscale is installed and logged in and then offering to configure `tailscale serve`. It SHALL show clear guidance when Tailscale is missing.
- **FR-20a**: *(rev 9)* **Settings → Remote access** and `gbi tailscale status` SHALL show a checklist, each line with its evidence and, when it fails, the fix:
  - Tailscale is installed, and running and logged in, with this machine's tailnet name;
  - a Serve rule points at go-build-it's own address (matched exactly, not by searching text);
  - the tailnet URL answers, and the answer comes from this go-build-it (same data folder), not another program or copy;
  - go-build-it is private: Funnel is off for its address;
  - no other service is on go-build-it's address.
  It SHALL also list everything else this machine serves, with the program behind each address when known.
- **FR-20b**: *(rev 9)* Setting up Serve SHALL NEVER replace another service's address. When the root of port 443 is free or already go-build-it's, setup MAY serve there without asking. When another service holds it, go-build-it SHALL name that service, propose its own HTTPS port, show the exact `tailscale serve` command, and run it only after the person confirms. **Stop serving** SHALL remove only go-build-it's own rules. Only the owner MAY set up or stop serving, and each change SHALL be recorded in the history.
- **FR-20c**: *(rev 9; rev 11)* When go-build-it is reachable and private, Remote access SHALL show its tailnet URL as a QR code that opens the mobile UI (`<tailnet URL>/m/`), with that address written under it, generated inside go-build-it, with a reminder that the phone must be signed in to the same tailnet. `gbi tailscale qr` SHALL draw the same code in a terminal.
- **FR-20d**: *(rev 9)* The doctor SHALL check remote access (`tailscale.serve`): skipped when go-build-it is not served on a tailnet, a failure when Funnel makes it public, and a warning when its rule exists but the URL does not reach this go-build-it.

### 4.3 Roles

- **FR-25**: The application SHALL provide these built-in roles. Each can be filled by an AI persona or a human:

| Role | Responsibility |
|---|---|
| **Project Manager** | Owns the roadmap, the task queue, prioritization, review gates and the flow of questions to humans. |
| **Architect** | Owns technical design, makes sure work follows the global standards, and produces design option sets. |
| **Developer** | Implements tasks. |
| **QA** | Writes and runs unit and end-to-end tests, and runs dress rehearsals and UAT. |
| **Reviewer** | Reviews completed work against requirements, standards and values. |
| **Git / Release Manager** | Handles all git and forge operations (see §9), so other personas don't spend tokens on them. |

- **FR-26**: Users SHOULD be able to define additional custom roles. *(rev 12)* These are **team roles** (the job someone does). What someone may do is set by **access roles** (FR-174), which SHALL also be customizable.

### 4.4 Users, sign-in and access control (rev 12)

go-build-it runs as **one shared server** for a team (decided 2026-11-12). Two ideas are kept apart: a **team role** (§4.3) is the job someone does on a project, filled by a person or an AI; an **access role** is what someone may do. A person can be a project's reviewer and hold Approver access. Three takes on the screens are in the [user management mockups](mockups/user-management.html). **Take 1, the admin console, was chosen on 2026-11-20 (Q13)**: People & access is a system area with Users, Groups and Roles, and each project has an Access tab with "Check access". Dev mode (§4.5) comes first, so multi-user work can be tried on one machine before sign-in exists.

#### Modes and sign-in

- **FR-160**: Every person SHALL be a **user**: a stable ID, display name, email (unique), status, and the time they were created and last seen. History events, comments, answers and sign-offs SHALL point at the user ID, so renaming someone does not rewrite the record.
- **FR-161**: go-build-it SHALL support **local accounts** first: a password (at least 12 characters, hashed with a memory-hard algorithm such as Argon2id, and checked against a list of known leaked passwords) and a **second step**, a passkey (WebAuthn) or a one-time code (TOTP). Whether the second step is required SHALL be a setting, per installation, with a stricter setting allowed for admins and approvers. Forgotten passwords SHALL be reset by an admin, or by email once email is configured.
- **FR-162**: go-build-it SHALL support **single sign-on with OIDC** (for example Entra ID, Okta, Google Workspace or Keycloak) as the next sign-in method; it is **required**, though the provider is not chosen yet. Settings SHALL take the issuer URL, client ID and secret, test the connection, and show the redirect URL. Users SHALL be matched by verified email. Group membership MAY come from a claim (FR-170). Local accounts MAY be turned off once OIDC works, keeping at least one break-glass account (FR-167).
- **FR-163**: An installation SHALL run in **solo mode** (one owner, today's behaviour, no sign-in from this machine) or **multi-user mode**. Switching to multi-user mode SHALL be an explicit action by the owner that first creates the owner's own account, and SHALL be recorded in the history. *(Rev 15:)* An installation set up as a **shared server** (for example `gbi service install --server` on Linux) SHALL start in multi-user mode; `gbi serve` on a workstation SHALL start in solo mode.
- **FR-164**: In multi-user mode, **loopback requests SHALL NOT be trusted by default**. A setting SHALL allow trusting loopback **as a named user** (typically a service account). Untokened loopback requests then act as that user, with that user's access and never more, and are recorded under that user's name. Settings SHALL warn that anything running on the machine, including agents, gets that access.
- **FR-165**: In multi-user mode, a **Tailscale identity** SHALL only identify a user that is linked to it (by an admin, or by the user after signing in). An unlinked Tailscale login SHALL get the sign-in page.
- **FR-166**: The **CLI** SHALL sign in with `gbi login` (a device-code flow in the browser) and keep a token that expires (default 30 days) and can be revoked. The **MCP server** SHALL use the same tokens.
- **FR-167**: **Agent sessions** SHALL get their own short-lived token, bound to the session and its task and ending with it, in place of the `X-GBI-Persona` header that any local process can send. In multi-user mode the header SHALL be ignored. A break-glass command run on the server (`gbi users recover`) SHALL restore an owner when every owner is locked out, and SHALL be recorded.
- **FR-168**: Web sessions SHALL expire (default 12 hours, 30 minutes idle; both settings), use secure, HTTP-only, same-site cookies, and be protected against cross-site request forgery. Repeated failed sign-ins SHALL lock the account for a while (default 5 tries, 15 minutes) and be recorded.

#### Users and groups

- **FR-169**: A user SHALL have one status: **Invited** (an invitation link, which expires), **Active**, **Suspended** (temporarily blocked, everything kept), **Locked** (after failed sign-ins; clears by itself or by an admin) or **Deactivated** (can't sign in; history stays attributed). Suspending or deactivating SHALL end that user's sessions and tokens at once. Every status change SHALL be recorded with who did it and why.
- **FR-170**: Admins SHALL invite users by email, choosing their first groups or project access. A user's page SHALL show their status, sign-in methods, projects and access roles (with where each came from), live sessions and tokens (each can be ended), and their access history. `gbi users …` SHALL do the same from the CLI.
- **FR-171**: go-build-it SHALL support **groups** (for example the stakeholder kinds: Product owners, Cybersecurity, DevSecOps, UI/UX, Transformation, Sponsors, Developers). Access SHOULD be granted to groups rather than to people one by one. With OIDC, a group MAY follow a claim from the identity provider. **Service accounts** (users that can't sign in interactively, only use tokens) SHALL be supported for automation and loopback trust.

#### Access roles and grants

- **FR-172**: Access SHALL be built from **permissions**. **Project permissions**: view; comment and mark up artifacts; answer questions; queue tasks and file findings; start agent work (spends money); edit artifacts; manage the team, tuning and members; approve gates (optionally limited to gate types); merge and release; project settings; manage project access. **System permissions**: manage users and groups; manage roles; manage archetypes and standards; server settings, remote access and updates; view all projects; run access reviews and export.
- **FR-173**: An **access role** SHALL be a named set of permissions with a scope (project or system). go-build-it SHALL ship built-in roles: **Owner** and **Admin** (system); **Viewer**, **Commenter**, **Contributor**, **Approver**, **Maintainer** and **Project admin** (project). Built-in roles can be copied but not edited.
- **FR-174**: Admins SHALL be able to create **custom access roles** from the start, as flexible as custom artifact tabs (FR-151): pick permissions, optionally start from another role, limit approval to gate types. Roles SHALL be versioned. Saving a role SHALL show who it affects, and SHALL be recorded.
- **FR-175**: Permissions that spend money (start agent work) or can't be undone (merge and release) SHALL be marked wherever roles are edited or granted.
- **FR-176**: Projects MAY define their own roles in the repository (for example `[roles]` in `docs/gbi.toml`), within a **ceiling** the server admin sets: a project file SHALL NOT be able to grant a permission the ceiling doesn't allow, and *who* holds a role SHALL be kept only in go-build-it, never in the repository. (Take 2. **Not planned**: Take 1 was chosen (Q13); roles live in go-build-it. Kept here in case a project needs roles in its repository later.)
- **FR-177**: A **grant** SHALL give a user, a group or an AI member an access role on one project, or on all projects. A grant MAY expire. Each project SHALL have a default for signed-in users without a grant (none, or a role), and MAY be **private**: hidden, name included, from people without access. One person SHALL be able to hold different roles in different projects. Projects that declare compatibility with each other SHALL NOT share access.
- **FR-178**: A project's access list SHALL show every grant with where it came from (direct, a group, the project default, an archetype) and SHALL answer **"can this person do this, and why?"** for any user, AI member and permission.

#### AI members, separation of duties and sign-off

- **FR-179**: **AI members** SHALL be subject to the same permission checks as people. Each AI member SHALL have an access role (by default from its archetype) that acts as its **ceiling**. When an AI member acts for a person, it SHALL get only the permissions both of them hold. By default AI members SHALL NOT answer questions, approve gates, change access, or merge and release, except Courier, which may merge and release when a person with that permission has signed off (Q16).
- **FR-180**: **Nobody signs off their own work**: the author of a change, and the person who directed the AI member that made it, SHALL NOT approve a gate on it. A project MAY require more than one approver.
- **FR-181**: Someone who opens a link to a project they can't see SHALL be offered **Request access** with a reason. Requests SHALL go to the inbox of the project's access managers (desktop and phone), where they can be granted with a role or declined. Private projects SHALL NOT reveal their name.
- **FR-182**: A gate type MAY name the groups or roles that must sign it off, in order or side by side, with how many from each (a **sign-off route**). go-build-it SHALL route the request to those people's inboxes and show the route's progress on the gate. *(This builds on the review gates of §10.3, FR-115–FR-118, and the sign-off workflow decided 2026-11-11.)*
- **FR-183**: A sign-off SHALL record who, when, which version of the artifact or release, and the comment. Changing what was signed SHALL withdraw the sign-off and say so.
- **FR-184**: A signer MAY **delegate** a step for a period (for example during leave); the delegation SHALL be recorded and shown on the route.

#### Enforcement and audit

- **FR-185**: Every API route SHALL declare the permission it needs, in the route table (FR-31), so the web UIs, CLI and MCP server are all checked in the same place. The UIs SHALL hide what someone can't do but SHALL NOT rely on hiding. A denied request SHALL say which permission is missing.
- **FR-186**: The history SHALL record sign-ins, failed sign-ins and sign-outs; status changes; grants added, changed, expired and removed; role changes; token creation and revocation; loopback-trust and sign-in setting changes; and denials of sensitive actions. *(A tamper-evident log stays out of scope; see §15.)*
- **FR-187**: An admin or auditor SHALL be able to run an **access review**: who can do what in each project, with findings to decide on (accounts unused for 60 days, direct grants, expiring access, people who could both build and sign off the same work, loopback trust), each kept with a reason or removed. The finished review SHALL be exportable (CSV) and recorded.
- **FR-188**: There SHALL always be at least one active Owner; the last one SHALL NOT be suspended, deactivated or demoted.
- **FR-189**: Temporary elevation MAY be requested (a role on a project for a set time, with a reason) and approved by someone who manages that project's access; it SHALL end by itself and be recorded. *(Later phase.)*

### 4.5 Dev mode (rev 13)

Dev mode is for evaluating and iterating on go-build-it itself: one person on one machine plays several users to try multi-user work, without sign-in.

- **FR-190**: go-build-it SHALL have a **dev mode**, turned on by starting the server with `gbi serve --dev` (or `dev = true` in config.toml). Dev mode SHALL be refused, with the reason, while go-build-it is reachable from other machines (external interfaces or Tailscale Serve, §4.2), and SHALL be shown in `gbi doctor` and Settings.
- **FR-191**: In dev mode a **dev bar** SHALL be fixed at the top of every desktop and phone screen, in a warning colour that no other part of the UI uses. It SHALL show the user you are acting as, and SHALL switch to any other user in one click (or `gbi dev as USER` from the CLI) without signing in. The switch SHALL apply at once to every open screen of that browser.
- **FR-192**: On the first start in dev mode go-build-it SHALL offer to create **two dev users** (an owner and a second person, for example `david` and `maria`) with their Assistants (§4.6), so multi-user work can be tried at once. Further dev users MAY be added from the dev bar.
- **FR-193**: Every action taken in dev mode SHALL be recorded as the user acted as, marked **dev** with the person who switched; and the dev bar SHALL show why the last request was allowed or denied (the permission and where it came from, FR-178), so access rules can be checked as you click.

### 4.6 The personal Assistant (rev 13)

- **FR-194**: Every user SHALL have a personal **Assistant**: an AI that works only with that user and operates go-build-it for them. It is not a project team member (§4.1): team members do the project's work, the Assistant talks with its person. Every side chat in the UI (the "Edit with" chat beside artifacts, archetypes and standards, the persona coach, Learn's helper) SHALL be a conversation with your Assistant, which makes the change or asks the right team member.
- **FR-195**: An **Assistant archetype** SHALL be defined system-wide (instructions, harness, model), like the other archetypes (FR-12). Every user's Assistant SHALL be made from it automatically. Each user SHALL be able to **tune their own Assistant** in their profile (added instructions, and the harness and model where the admin allows a choice); archetype changes SHALL reach Assistants as FR-12b describes.
- **FR-196**: A user's Assistant SHALL be **always running while the user is signed in**: it starts at sign-in (or when dev mode switches to the user), runs in its own herdr pane in that user's workspace, and stops after the user signs out or has been idle for a set time (default 2 hours). Its conversation SHALL be resumed next time.
- **FR-197**: The Assistant's **status** SHALL be shown on every screen (desktop top bar and phone header): starting, ready, working, waiting for you, or offline, with its context use. Clicking it SHALL open the Assistant panel; a stopped or blocked Assistant SHALL say why (FR-205).
- **FR-198**: The Assistant SHALL act with **its user's permissions** and never more (FR-179). It MAY read anything its user can, and queue work, ask members and draft changes. It SHALL NOT submit a decision on its own: answers, verdicts, gate sign-offs, choices between options and access changes SHALL be drafted for the user to confirm. Its actions SHALL be recorded as "Assistant for USER".
- **FR-199**: The Assistant SHALL use go-build-it through the CLI and MCP server with a token bound to its user and to itself (FR-167), so its access can be checked and ended like any token.

### 4.7 Setting up in the browser (rev 15)

Nothing has to be installed or configured on a person's own machine to use go-build-it: they sign in to the server, and the server sets up their Assistant with them, in the browser. Mockups: [setup wizard, W1–W4](mockups/user-management.html#wizard); the vision's [How people reach it](vision.html).

- **FR-219**: On first sign-in (and on first start in solo mode), go-build-it SHALL offer a **setup wizard**, "Set up your Assistant": (1) choose a harness (Claude Code, Antigravity, Pi, or another enabled one), each with a line on what it needs; (2) sign it in (FR-220), or give an API key (FR-221); (3) **check** it: go-build-it confirms the harness is signed in and sends a short test prompt, which also catches a model that stops at a plan or usage menu (FR-205), and the person picks a model that works; (4) the Assistant starts (FR-196). The same wizard SHALL add another harness later, from the profile.
- **FR-220**: To sign a harness in, the server SHALL start it for that person (in their own harness configuration folder, and in their workspace where workspaces exist, FR-207) in a herdr pane, and show its **terminal in the page**, beside plain steps. When the terminal shows a sign-in link, the page SHALL also show it as a button; when the harness asks for a code, a paste box SHALL type it into the terminal. Success SHALL be detected by checking the harness (FR-07b), not by scripting its dialog: the person completes the harness's own sign-in.
- **FR-221**: Where a harness's sign-in can't complete through a web terminal (for example one that returns to a local address on the machine running it), the wizard SHALL say so and offer that harness's **API key** instead (FR-201). A spike SHALL establish, for each supported harness, which way works.
- **FR-222**: The wizard SHALL be **skippable**: someone who only answers questions and signs off can use go-build-it without an Assistant. Until setup is complete, go-build-it SHALL show it: the Assistant chip reads "Set up" (FR-197), the Projects home shows a **Finish setting up** card with the steps left, and the profile shows the same checklist. Each SHALL reopen the wizard at the first unfinished step. Dismissing the card SHALL hide it, never the chip.
- **FR-223**: When a harness's sign-in expires or stops working, the person SHALL see it on the Assistant chip and in their inbox, with **Fix**, which reopens the wizard at the sign-in step.
- **FR-224**: Only the person themself SHALL be able to open the terminal of their own harness during setup. Admins SHALL see whether each user's setup is complete and working, never their sign-in screens or keys.
- **FR-225**: In Shared mode, an admin SHALL set up the server's harnesses through the same wizard in **Settings → Harnesses**, and users skip the harness steps.
- **FR-226**: The **CLI** MAY be pointed at a server: `gbi context add NAME URL`, `gbi login`, `gbi context use NAME`. Every command that changes something SHALL name the server it acts on. `gbi mcp` SHALL use the current context, so a person's own AI tool can work with the server under their permissions. Nothing else SHALL need setting up on their machine.
- **FR-227**: From the CLI, a person SHALL be able to watch and take over their agents' terminals on the server (`gbi sessions attach`), with the same rules as the web terminal (FR-66, FR-214).
- **FR-228**: Setting up and fixing the Assistant SHALL be easy to find wherever someone looks: the first sign-in, the top-bar Assistant chip, the **Finish setting up** card on the Projects home, the profile (**My Assistant**), **Settings → Harnesses** for admins, the phone's home and More screens, the ⌘K palette ("Set up Assistant"), the first Learn lesson, and the docs.

## 5. User interfaces

- **FR-30**: Everything that can be done in the web UI SHALL also be possible through the CLI, so AI agents can operate the application. (AI-native requirement.)
- **FR-31**: The application SHALL be API-first. The web UIs and the CLI SHALL be clients of the same HTTP API. The API SHOULD be described by an OpenAPI spec.
- **FR-32**: CLI commands SHALL support machine-readable output (`--json`).
- **FR-33**: The application SHOULD expose an MCP server so external agents can drive it.
- **FR-34**: The application SHALL provide a **desktop web UI** designed for laptop and desktop screens.
- **FR-34a**: *(rev 10)* In the desktop UI, the logo and the application title SHALL link to the landing page (the project catalog), by mouse and by keyboard.
- **FR-35**: The application SHALL provide a separate **mobile web UI** designed for phones. It SHALL put the most common mobile tasks first: answering questions, reviewing and giving feedback, checking project status, and approving review gates.
- **FR-35a**: *(rev 11)* A phone (a narrow touch screen) that opens a desktop address SHALL be taken to the matching mobile screen (for example `/questions/Q-1` to `/m/q/Q-1`, a project to its mobile status, anything without a mobile screen to `/m/`). **More → Desktop site** SHALL switch that device to the desktop UI until it chooses **Mobile UI** again; the choice SHALL be kept per device. A narrow desktop window without touch SHALL keep the desktop UI. On narrow screens the desktop UI SHALL keep its navigation behind a ☰ button that opens the sidebar as a drawer (closed by a choice, a tap outside, or Esc), and desktop pages SHALL fit the screen's width, with wide content scrolling inside the page.
- **FR-36**: User interactions SHALL feel interactive and dynamic, like working with a team. Updates (agent progress, new questions, status changes) SHALL be pushed to the UI live, without manual refresh.

## 6. Projects

### 6.1 Project catalog

- **FR-40**: The landing view SHALL be the project catalog, showing all managed projects as quick-view cards.
- **FR-41**: At a glance, each card SHALL show:
  - **What the project is**: name, a one-line purpose, and its key values or goal.
  - **Where it stands**: lifecycle state, current roadmap phase, active sessions, open questions (blocking count highlighted), pending review gates, last activity, and CI status.
  - **What has shipped**: the latest stable release (version, release date, and a link to its release notes), plus how many commits have landed since that release.
  - **Where to find it**: links to the project website (see FR-44), the repository on GitHub or GitLab, and the project's embedded documentation.
  - **Other useful information**:
    - open PRs/MRs and open issues
    - the assigned team
    - a warning for unpushed work (see FR-94)
    - test status from the latest CI run
- **FR-41a**: Clicking a card's release or website link SHALL open the target directly. Cards SHALL show a clear placeholder (e.g. "No stable release yet", "Website not published") when information doesn't exist yet, rather than leaving gaps.
- **FR-42**: The catalog SHALL support search, filtering (by state, team, or needs-attention) and sorting, because many projects will be managed at once.
- **FR-43**: Projects SHALL have a defined lifecycle. Initial states: *Discovery → Requirements → Design → Implementation → Review → Released → Maintenance*, plus *Paused* and *Archived*.
- **FR-44**: Each project SHALL have a website hosted on GitHub Pages. GitLab-hosted projects SHOULD use GitLab Pages instead. The Git / Release Manager role SHALL publish or update the website as part of each stable release. The website URL SHALL be stored with the project and shown on its card.

### 6.2 Project artifacts

- **FR-45**: The team SHALL be able to develop the following with the user for each project:
  - requirements documents
  - mockups
  - roadmaps
  - a record of the project's **values**
- **FR-45a**: Each artifact SHALL have an **Edit** action that opens the artifact editor. In the editor the user SHALL be able to:
  - chat with an AI agent (e.g. the Project Manager or Architect persona) that proposes changes, shown as a highlighted diff the user accepts or rejects
  - edit the content directly in a rich editor: WYSIWYG for prose, or a code-editor style Markdown editor (syntax highlighting, preview, outline). A plain text area is not sufficient.
  - save, which creates a new revision. The revision SHALL be committed to the repository by the Git / Release Manager and shown in the artifact's revision history.
- **FR-45b**: Roadmap items (milestones and review gates) SHALL open a detail page showing: description and scope, tasks with status, linked requirements, gate criteria, assigned team members, and history.
- **FR-46**: Project artifacts SHOULD be stored in the project's git repository (e.g. under `docs/`) so they are versioned with the code. Operational state (queues, sessions, history) SHALL live in the application database.

### 6.3 Project documents (rev 7)

Every project keeps the same kinds of documents, so people, AI members and gbi can find them without guessing. See [ADR-014](decisions/ADR-014-project-documents.md).

- **FR-140**: A project's **Artifacts** tab SHALL have sub-tabs for its documents. The standard set is **Requirements**, **Mockups**, **Plan**, **Tracker**, **Compare** and **Support files**; which of them a project shows, in what order, and any others, is decided by `docs/gbi.toml` (FR-151).
- **FR-141**: Requirements, Mockups, Plan and Tracker SHALL each open one **entry point**: an HTML file under `docs/`, set explicitly (`gbi artifacts entry <project> <role> <path>`, or **Set as entry point** in Support files). go-build-it SHALL NOT pick an entry point by itself. Several sets of one kind (e.g. web and phone mockups) SHALL be handled inside the entry HTML, by linking to them.
- **FR-142**: Every file under `docs/` SHALL be reachable from **Support files**, with its labels. A file gets labels only when someone gives them: `gbi artifacts add <project> <path> --label …` (a label is required) or `gbi artifacts label`, or the same action in the UI. A label on a folder covers the files in it. The labels are a fixed list: `requirements`, `mockup`, `plan`, `tracker`, `screenshot`, `decision`, `research`, `asset`, `support`. go-build-it SHALL NOT infer labels; files without one SHALL be listed as **Unlabelled**.
- **FR-143**: Entry points and labels SHALL be stored in the repository in `docs/gbi.toml`, and every change SHALL be committed by Courier and recorded in the history.
- **FR-144**: HTML artifacts SHALL be shown inside go-build-it in a sandbox: scripts run, files they reference load from the repository by relative path, and the page cannot use go-build-it's API. The viewer SHALL follow links between documents: a link to another tab's entry point switches to that tab, a Markdown file opens rendered, the `#anchor` is kept, and the address bar records where you are, so it can be shared.
- **FR-145**: go-build-it SHALL serve a helper that lets an HTML page include Markdown (`<gbi-md src="requirements.md">`), so prose can be written in Markdown while the entry point stays HTML. Requirement lines (`**FR-07**:`) and headings SHALL become anchors.
- **FR-146**: Documents SHALL use stable IDs: requirement IDs (`FR-nn`, `NFR-nn`), and mockup views marked `data-view="v03"` with optional `data-req` and `data-route`. Tasks reference requirements in their `reqs` and may name views (`v03`) in their text. A **traceability** view and `gbi trace list` SHALL show each requirement's views, tasks and real screens, and the gaps: a requirement with no task, a view that names an unknown requirement, and a view whose real screen has drifted or was never reviewed.
- **FR-147**: **Compare** SHALL pair each mockup view with the screenshot labelled `screenshot` and named after the view (`v03.png`), side by side or as an overlay with a slider. For each view the person SHALL be able to **accept** the real screen or **file a task**, with a note. Decisions SHALL be stored in `docs/compare.json` with the screenshot's checksum, so a view returns for review when its screenshot changes. Its state is one of: no screenshot, not reviewed, accepted, drifted, task filed.
- **FR-148**: The tracker entry point SHALL read a snapshot of the project's milestones and tasks (`docs/tracker/tracker.json`), which Courier writes from go-build-it's database on request (`gbi artifacts snapshot`). go-build-it's database SHALL stay the source of truth.
- **FR-149**: `gbi artifacts init <project>` SHALL create the standard set from templates without overwriting existing files: entry points for requirements, mockups, plan and tracker, Markdown sources, a screenshots folder and `docs/gbi.toml`.
- **FR-151**: *(rev 8)* `docs/gbi.toml` SHALL control the sub-tabs completely:
  - `[entries]` maps a role to its HTML entry point. A role is any short lowercase name: the standard `requirements`, `mockups`, `plan`, `tracker`, or the project's own (e.g. `vision`). Each role with an entry point gets a sub-tab, in the order the entries are written. A role without an entry point has no tab, so a project without mockups has no Mockups tab. **Compare** appears only with a mockups entry; **Support files** always.
  - An optional `tabs = [...]` lists the sub-tabs in order, and only those are shown; `compare` and `files` name the built-in ones.
  - An optional `[titles]` renames tabs.
  - go-build-it SHALL keep `tabs`, `[titles]` and the order of `[entries]` when it rewrites the file. `gbi artifacts check` SHALL report a tab listed without an entry point, an entry point that is missing or not HTML, and unknown labels, and SHALL NOT report standard roles a project left out. `gbi artifacts init` SHALL create only the documents for the roles the project uses (those in `tabs`, or the standard set).
- **FR-150**: `gbi artifacts check` SHALL report missing or invalid entry points, unlabelled files, labels on missing files, broken links in labelled HTML, and IDs that point to nothing. A global standard, **Project documents**, SHALL describe the layout, labels and IDs, and the agent guide SHALL tell AI agents to add files with `gbi artifacts add`.

## 7. Global standards

- **FR-50**: The application SHALL maintain a set of global standards that apply to every project. Examples:
  - standard backend stack
  - standard frontend stack
  - unit tests required
  - end-to-end tests required
  - documentation requirements
- **FR-50a**: *(rev 9)* A new installation SHALL start with go-build-it's starter standards, shipped in the binary from `docs/standards/` (access control, AI layer security, code design, development loop, the DoD application security, ATO readiness, audit logging and secure supply chain standards, Go backend, operational resilience, planning and alignment, React frontend, software hardening), plus the built-in Project documents standard (FR-150). Markdown files in `~/.go-build-it/standards/` SHALL be used instead when present. An installation that already has standards SHALL NOT be reseeded.
- **FR-51**: Standards SHALL be authored as Markdown and managed in the UI and the CLI. They SHALL be edited in the artifact editor (FR-45a), so an AI agent can help draft or change them.
- **FR-52**: A project MAY override or extend a global standard. Every override SHALL be recorded with its reason.
- **FR-53**: Standards SHALL be supplied to every agent session automatically, for example by generating the harness's instruction file (`CLAUDE.md`, `AGENTS.md` or equivalent), or by injecting them into the session context.
- **FR-54**: The Reviewer role SHALL check completed work against the applicable standards.

## 8. Agent harness orchestration

### 8.1 Harness integration

- **FR-60**: The application SHALL run every agent harness interactively in a herdr pane of the gbi herdr session (FR-07), at the direction of the development team. herdr SHALL be the only process host: there SHALL be no tmux, direct-PTY or headless (`-p` / `stream-json`) execution path.
- **FR-61**: Harness support SHALL be implemented through a harness profile that describes, for each harness:
  - its herdr agent kind and launch arguments (model, permission mode, flags)
  - how to resume a conversation (e.g. `claude --resume <id>`)
  - where its transcript lives and how to read events, messages and context-window usage from it
  - how it reports rate or usage limits
  The application SHALL start the harness with `agent start`, send work with `agent prompt`, follow state through herdr events, interrupt with `agent send-keys`, and detect exit through pane events.
- **FR-62**: Initial adapters: **Claude Code** (Anthropic subscription), **Antigravity** (Google / Gemini subscription) and **OpenCode** (open-source harness that works with multiple model providers). Pi-agent and others SHOULD be addable through the same interface.
- **FR-63**: Harnesses and their settings (binary path, model, flags, permission mode) SHALL be configurable in the UI and the CLI.
- **FR-63a**: *(rev 10)* go-build-it SHALL discover the models each harness offers and offer them wherever a model is chosen: an archetype's default model and a member's model override. Harnesses with a list command SHALL be asked (Antigravity `agy models`, Pi `pi --list-models`, OpenCode `opencode models`); Claude Code has none, so go-build-it SHALL ship its aliases and current model IDs. The choice SHALL be a searchable list that still accepts any typed name, with a warning when the name is not in the harness's list. Lists SHALL be cached per harness, refreshed in the background once a day or on request, and a failed refresh SHALL keep the last good list. `gbi harnesses models <harness>` SHALL return the same list.
- **FR-66**: The live view of an AI team member SHALL offer two modes:
  - a **structured view**: events and messages read from the harness's transcript (edits, commands, test results, messages)
  - a **terminal view**: the member's herdr pane streamed live to the desktop and mobile UI (herdr `terminal session observe`), with the option to take control (`terminal session control`) and type, resize and use keys. Several viewers can watch at once; one controls at a time.
- **FR-67**: The gbi herdr session SHALL be organized so that opening it shows everything go-build-it is doing:
  - one herdr workspace per project, with each AI team member's worktree grouped under it
  - each member's pane labelled with its name, role and current task, plus small status values such as context usage (herdr pane metadata)
  - each project workspace showing a summary such as "2 working · 1 needs you" (herdr workspace metadata)
  - review gates and questions that need the human raised as herdr notifications
- **FR-64**: Each AI team member SHALL work in its own long-lived git worktree, so concurrent members don't interfere. Each task SHALL get a fresh branch in that worktree. The application SHALL keep creating worktrees itself and SHALL ask herdr to open them grouped under the project workspace.
- **FR-68**: The application SHALL treat herdr's agent state as the source of truth for whether a member is working, blocked, done or idle. A **blocked** member SHALL appear in the UI as needing the human, linked to its live terminal view. The application SHALL NOT try to understand harness dialogs from the screen, except the folder trust prompt (FR-07f). It SHOULD avoid permission prompts through permission modes and per-project allowlists, and MAY use harness hooks (e.g. Claude Code's permission hooks) to offer structured approvals.
- **FR-69**: Each AI team member's conversation SHALL continue across tasks until it is rotated (§8.2). After a herdr or machine restart, the application SHALL find each member's pane again by its conversation ID and SHALL resume or relaunch it.
- **FR-65**: Cost is limited by the user's AI subscriptions, so no budget controls are needed. The application SHALL detect subscription rate or usage limits reported by a harness. It SHALL then pause the affected work and resume it automatically when the limit resets.

### 8.2 Context management and smart handoffs

By default, an AI team member's conversation relies on the harness's built-in auto-compaction when the context window fills. Smart handoffs (FR-72–FR-74) rotate the member to a new conversation instead; they are **experimental**, off by default and enabled in Settings, globally or per project. FR-70, FR-71 and FR-76 apply in both modes.

- **FR-70**: The application SHALL prepare each session's starting context deliberately: the task, the relevant requirements, standards, values, and the incoming handoff (if any). It SHALL NOT dump all of the project's history.
- **FR-71**: The application SHALL monitor each AI team member's context-window usage, read from its harness transcript, and SHALL show whether the value is measured or estimated.
- **FR-72**: *(Experimental.)* When smart handoffs are enabled and usage reaches a configurable threshold (e.g. 70–80%), the application SHALL step in and start a smart handoff **before** the harness auto-compacts. When they are disabled, the application SHALL record each compaction event in the history.
- **FR-73**: A handoff SHALL be a curated briefing, written as if handing responsibility to another person. At minimum it SHALL contain:
  - objective and definition of done
  - current state and what has been completed
  - decisions made and why
  - open problems, risks and dead ends to avoid
  - specific next steps
  - pointers to the relevant files and artifacts (references, not copies)
- **FR-74**: The next session SHALL start from the handoff and FR-70's prepared context. Every handoff SHALL be stored in the history.
- **FR-76**: A member's lasting knowledge SHALL live in go-build-it (persona, project values, decisions, notes from finished tasks), not only in its conversation. The application SHALL rotate a member to a new conversation when the rotation policy says so (see Q8), starting it from FR-70's prepared context and the member's notes.
- **FR-75**: Tasks SHOULD be sized so most of them fit in one session. The Project Manager role SHOULD split tasks that repeatedly need handoffs.

### 8.3 Task queues

- **FR-80**: Each project SHALL have a persistent task queue that can manage large numbers of tasks.
- **FR-81**: Queues SHALL support:
  - priorities
  - dependencies between tasks
  - concurrency limits per project and globally
  - retries
  - assignment of tasks to roles or to specific AI team members
- **FR-82**: Queue state SHALL survive application and machine restarts. Work interrupted by a restart SHALL continue in the member's resumed conversation (FR-69), or from its latest handoff if the conversation cannot be resumed.

### 8.4 Upstream tracking

- **NFR-30**: Because go-build-it depends on herdr, it SHALL keep a record of the herdr and Ovis (a herdr web client) source revisions last reviewed, what changed, and what go-build-it did about it (`docs/upstream/`).
- **NFR-31**: A scheduled review SHOULD run at least weekly. It compares the latest herdr and Ovis sources, herdr's API schema (`herdr api schema --json`) and changelogs with the recorded revisions, and opens an issue or PR when something affects go-build-it.
- **NFR-32**: go-build-it's herdr client types SHALL be generated from herdr's published API schema, so an incompatible change fails the build. The supported herdr version range SHALL be declared in one place and tested in CI.

### 8.5 Harness credentials and warm harnesses (rev 13)

- **FR-200**: A system setting SHALL choose how harnesses are paid for: **Shared**, where every harness runs on the server's own sign-in or key, or **Per user**, where a user's Assistant and the agent work that user starts run on that user's own credentials. Both SHALL be supported equally well. *(Rev 15:)* In multi-user mode the default SHALL be **Per user**, and every person sets up their own through the setup wizard (FR-219); Shared is an admin option, set up once by an admin through the same wizard.
- **FR-201**: A harness credential SHALL be one of: a harness that is **already signed in** before go-build-it starts (in Per user mode, signed in inside that user's own harness configuration folder, guided step by step in a herdr pane); or an **API key** for the harness, optionally one per model. Keys SHALL be stored encrypted, never shown again after saving, and never written to logs, the history or a repository (NFR-22).
- **FR-202**: Settings, and each user's profile in Per user mode, SHALL show which credential each harness and model uses, and SHALL test it with the doctor (FR-07b) without starting project work.
- **FR-203**: In Per user mode, agent work SHALL run on the credentials of the user who started it. Automatic dispatch SHALL use the credentials of a user named for the project; with none named, it SHALL not start paid work.
- **FR-204**: go-build-it SHALL keep **warm harnesses** started ahead of need: every signed-in user's Assistant, and the members of projects with queued or running work, within limits set in Settings (how many, and how much memory). Others SHALL start on demand, and idle ones SHALL stop after a set time. Settings SHALL show each warm harness with its state and memory.
- **FR-205**: When a harness stops at a screen go-build-it does not answer (a model or plan menu, a sign-in, a permission), the reason the person sees SHALL be that screen's text, and the doctor's prompt tier SHALL check that the configured model starts without such a screen.
- **FR-206**: A short-lived harness used for one question (a reviewer's assessment, release notes) SHALL get the same care as a member: folder trust answered (FR-07f), and a prompt that the harness dropped while starting sent again.

### 8.6 Per-user workspaces on a Linux server (rev 14)

On one shared Linux server, people's AI agents run whatever code their projects bring. Each person therefore works in their own isolated **workspace**. The proposed design (ADR-017) pulls standard images with go-containerregistry and runs them with gVisor, with no daemon; runc is the faster level for trusted projects and Firecracker a later one where KVM exists. A spike proves it first (Q18–Q20).

- **FR-207**: In multi-user mode on Linux, each user SHALL have their own **workspace**: an isolated environment that holds their Assistant (§4.6), the agent sessions they start, the worktrees those sessions use, their harness configuration and credentials (FR-201), and their own herdr session. Nothing in a workspace SHALL be able to see or reach another user's workspace, files, processes or credentials, or the server's own data folder.
- **FR-208**: Workspaces SHALL run behind **one runtime interface** in go-build-it (create, start, run a command, attach a terminal, stop, remove, report use), so the runtime can be chosen per installation. At least two levels SHALL be supported: **container** (Linux namespaces and cgroups, rootless where the host allows) and **strong isolation** (a user-space kernel such as gVisor, or a microVM such as Firecracker where KVM is available). An admin MAY require strong isolation for a project, for example one that builds untrusted repositories.
- **FR-209**: go-build-it SHALL NOT require the Docker Engine. Workspaces SHALL start from standard **OCI images**, so existing Docker images work. go-build-it SHALL provide a default workspace image with git, common toolchains, herdr and the supported harnesses, and a project MAY name its own image from a registry.
- **FR-210**: Each workspace SHALL have **limits** (CPU, memory, disk, processes) set by the admin for everyone, per group or per user, and its use SHALL be shown in Settings, on the user's profile and in the dev bar (FR-191).
- **FR-211**: A workspace's **network** SHALL by default reach only what the admin allows (the git forges, package registries and the harness providers' APIs). It SHALL NOT reach other workspaces or services on the host, except go-build-it's API through a channel scoped by its user's token (FR-199).
- **FR-212**: **Code leaves a workspace only through review.** A workspace gets its own clones or worktrees of the projects its user works on; changes leave as commits that Courier pushes and that go through review (§9, §10.2). One user's workspace SHALL never write into another's files or the server's project clones.
- **FR-213**: Agent work SHALL run in the workspace of the user who started it, on that user's credentials (FR-203). Work started by automatic dispatch SHALL run in the workspace of the user the project names for it.
- **FR-214**: Watching and taking over SHALL work across the boundary: go-build-it reaches each workspace's herdr through a socket only it can open, and the terminal view, session output and blocked detection (FR-66, FR-68) SHALL work as they do today. A person SHALL be able to watch another user's agent only if they may view that project (FR-172), and take control only of their own.
- **FR-215**: A workspace SHALL be **created** when its user first signs in (or is switched to in dev mode), **started** when they sign in or have work to run, **stopped** when idle (with the warm-harness rules, FR-204), and **kept** between sessions (its home folder persists). An admin MAY reset or remove a workspace, exporting or deleting its data. Each of these SHALL be recorded in the history.
- **FR-216**: In solo mode, and on macOS and Windows, go-build-it SHALL keep running everything on the host as today. The doctor SHALL report whether a Linux host supports the chosen runtime (user namespaces and subordinate IDs, cgroups v2 delegation, KVM for microVMs) and SHALL explain each gap, as it does for herdr.
- **FR-217**: A workspace SHALL be inspectable by its user: a terminal in it from the UI (desktop and phone) and from `gbi workspace shell`, and its image, limits, use and network rules shown on their profile.
- **FR-218**: Updating the workspace image SHALL be done like a go-build-it update (FR-06c): staged, applied when the workspace is idle, and reversible.

## 9. Git, GitHub and GitLab workflow

- **FR-90**: The application SHALL work with both GitHub (`gh`) and GitLab (`glab`), and SHALL detect which one each project uses and whether its CLI is authenticated.
- **FR-91**: The application SHALL enforce a defined development workflow. The workflow SHALL be defined as a global standard (*Git workflow*, §7) that projects may override per FR-52. For example: branch per task → commits → push → PR/MR → review → merge. Settings only hold forge connections and authentication.
- **FR-92**: All git and forge operations SHALL be carried out by the **Git / Release Manager** role, including:
  - branching and committing
  - pushing
  - opening PRs/MRs
  - reading CI status
  - merging
  - tagging releases
- **FR-93**: Deterministic git operations (e.g. push, open PR, read CI status) SHOULD be run directly by the application rather than through an LLM session, to save tokens. The Git / Release Manager persona is responsible for them and calls in an agent only when judgment is needed, such as resolving merge conflicts or writing release notes.
- **FR-94**: The application SHALL make sure completed work is pushed to the remote. Unpushed work SHALL be shown as a warning on the project card.

## 10. Human interaction

### 10.1 Questions

- **FR-100**: Team members SHALL be able to send questions to human team members through a dedicated channel.
- **FR-101**: Each question SHALL be marked **blocking** (the asking work pauses until it is answered) or **non-blocking** (work continues on a stated assumption, which the human can later confirm or correct).
- **FR-102**: Each question SHALL carry everything needed to answer it without digging:
  - the question itself
  - why it matters
  - the related task and requirement IDs
  - the asker's recommendation, if it has one
  - attached artifacts, such as mockups, diffs, screenshots, logs or option sets
- **FR-102a**: Attachments SHALL open in an in-app viewer suited to their type: HTML mockups rendered live (with desktop and mobile viewport sizes), images with zoom, diffs side by side, and JSON or logs with syntax highlighting.
- **FR-103**: Questions MAY offer structured answers (choices, yes/no, pick-an-option) as well as free text.
- **FR-104**: A **Questions dashboard** (on desktop and mobile) SHALL list open questions across all projects. It SHALL put blocking questions first and SHALL let the user answer each question in place, with its context, so they can get through them quickly.
- **FR-105**: Answers SHALL be sent back to the asking session or task automatically. Answers that reveal a project value SHALL be offered for recording in the project's values (§10.4).
- **FR-106**: Email and other out-of-app notifications are out of scope for now. In-app live updates (FR-36) are in scope.
- **FR-107**: Wherever the user writes an answer, feedback or chat message, they SHALL be able to dictate it by voice (speech-to-text) as well as type it. The transcript SHALL be editable before sending.

### 10.2 Feedback on completed work

- **FR-110**: The application SHALL provide an easy way to give feedback on completed work. The verdicts are: approve, request changes, or reject. Comments SHALL attach to a specific artifact, screen or diff location.
- **FR-111**: Feedback SHALL go into the context of the follow-up task automatically, and SHALL be recorded in the history.
- **FR-112**: The review of completed work SHALL be an assessment, not primarily a code diff. It SHALL show:
  - the approved mockup next to the built result (screenshots captured by the e2e tests)
  - a summary of the tests run: unit and end-to-end, passed, failed and coverage
  - an assessment against the global and project standards (a standard that does not apply to the change is marked so, with the reason, and is not counted)
  - an assessment against the feedback given during mockup reviews and dress rehearsals: addressed or not
  - an assessment against the requirements in scope: met, partly met or not met, with evidence
  - a link to the pull or merge request, where the code diff remains available
- **FR-112a**: *(rev 13)* The reviewer SHALL be given the whole change (the task, the diff since the branch left the main line, the worktree to read) and the tests go-build-it ran. The review SHALL list its **evidence gaps** (no tests ran, tests failed, requirements not met or not assessed, feedback not addressed, a mockup without a screenshot). No verdict SHALL be selected in advance, and approving with gaps SHALL need a stated reason.

### 10.3 Review gates and dress rehearsals

- **FR-115**: Roadmaps SHALL support **review gates**: defined points where work pauses for human review and feedback before going on.
- **FR-116**: A review gate MAY be a **dress rehearsal**. The application walks the human step by step through every feature or scenario in scope. It can do this as a tabletop exercise before implementation (using mockups and scenarios) or as a UAT run after it (using the working application).
- **FR-116a**: Mockups and screens shown in option sets, rehearsals and reviews SHALL open in a large viewer (full screen, with desktop and mobile viewport sizes), because a desktop-sized mockup does not fit inside a panel of the desktop app.
- **FR-117**: During a dress rehearsal, the human SHALL be able to give feedback on each step, and SHALL be able to talk with the agents live to request changes or improvements as the rehearsal goes on.
- **FR-117a**: More than one reviewer, human or AI persona, SHALL be able to record findings on the same rehearsal. Each finding SHALL be attributed to its author.
- **FR-118**: Rehearsal outcomes (feedback per step, changes requested, approval) SHALL be recorded. A gate SHALL NOT be passed until the human explicitly approves it.

### 10.4 Exploratory design and values

- **FR-120**: For significant design or implementation decisions, the team SHALL present an **option set** of 2–3 alternative approaches, not a single answer.
- **FR-121**: Each option SHALL include a summary and tradeoffs, and SHALL include a visual artifact where relevant (an HTML mockup or an image).
- **FR-122**: The human SHALL be able to choose one option, or combine parts of several options and explain why. Choices and reasons SHALL be recorded.
- **FR-123**: The application SHALL keep a living **problem definition** and a set of **values** (guiding principles) for each project. Both SHALL be refined from the human's choices, answers and feedback.
- **FR-124**: Future option sets SHALL be generated with the recorded values in mind, so options get closer to what the human wants as understanding grows.

## 11. Observability and self-improvement

- **NFR-10**: The application SHALL keep a complete, searchable history of what happened, when, why, and who or which persona did it. This includes:
  - session transcripts
  - handoffs
  - task state changes
  - questions and answers
  - feedback and design choices
  - git operations
- **NFR-11**: It SHALL always be possible to determine what the system is currently doing and why. Live views SHALL show active AI team members, their herdr state, their tasks, and their context usage.
- **NFR-12**: Records SHALL be structured so AI agents can analyze the application's own performance. Examples:
  - handoff quality
  - rework rates
  - question volume
  - time per task
- **NFR-14**: Every history event SHALL open a detail view suited to its type: session transcript or log, handoff, compaction, question and answer, review, rehearsal finding, option choice, git operation. Actions taken by humans SHALL be recorded and viewable in the same way as agent actions.
- **NFR-13**: Agents SHALL be able to use that analysis to propose and make improvements to go-build-it itself, such as prompts, persona definitions, handoff templates or code. These changes go through the normal review workflow.

## 12. Built-in documentation

- **FR-130**: Documentation SHALL be embedded in the application, searchable, and linked from the relevant places in the UI.
- **FR-131**: Documentation SHALL be fully illustrated with screenshots.
- **FR-132**: Documentation SHALL include a dress-rehearsal / UAT walkthrough that covers every feature, step by step, for every role.
- **FR-133**: The UAT walkthrough SHOULD be written as automated end-to-end scripts (e.g. Playwright) that both verify the features and regenerate the documentation screenshots, so screenshots stay current.

### 12.1 Learning go-build-it

- **FR-134**: go-build-it SHALL have a **Learn** area (desktop `g l`, and on the phone under More) with a series of short lessons that together cover every feature. Each lesson SHALL have:
  - a goal and an estimated time of 5–10 minutes
  - numbered steps
  - one to three **Pro tip** callouts
  - links to the how-to pages it builds on

  A lesson SHALL open as a coach panel docked beside the app, so the user follows the steps in the real UI. A step's **Show me** button SHALL go to the right view and highlight the control it names.
- **FR-135**: Lessons SHALL know when a step is done. A step is checked off from go-build-it's own history events (e.g. a task created in the practice project after the lesson started) or from the user visiting a view. The user MAY also tick it off. Progress SHALL be kept per user, so a lesson can be left and resumed. Lessons and progress SHALL also be available from the CLI (`gbi learn …`).
- **FR-136**: Lessons SHALL run in a **practice project** by default. It is created on request with its own small git repository under `~/.go-build-it/practice/` and a starter team whose AI members use the scripted fake harness, so practice uses no AI subscription and never touches the user's real repositories. It SHALL be clearly labeled as practice everywhere it appears, and the user SHALL be able to reset or remove it. The last lesson SHALL show how to move to a real project.
- **FR-137**: A lesson and a cookbook page SHALL teach controlling go-build-it **from the user's AI harness**, with Claude Code as the main example and notes for Antigravity, Pi and OpenCode:
  - the two ways to connect: running `gbi` commands, and the MCP server (`gbi mcp`), and how to choose between them
  - examples that each show what the user asks the harness, the `gbi` commands the harness runs, and what then appears in the UI. Terminal output SHALL be shown as text transcripts, and UI results as screenshots.
  - `gbi agent guide`, which prints a guide for AI agents (the workflow, the commands, and the rules such as asking before starting paid work) that the user can install as a harness skill or `AGENTS.md`
- **FR-138**: The docs SHALL include **How go-build-it thinks**, short explanation pages on why it is designed the way it is: herdr as the only backend, persona archetypes and project members, one worktree per member, gates and phases, questions instead of guesses, the history as an audit trail, and one route table for UI, CLI, API and MCP. Each page SHALL link the ADRs behind it.
- **FR-139**: The lessons SHALL be tested so they stay current:
  - Every `gbi` example in the lessons and the cookbook SHALL be run against a demo instance by an automated test, which fails when an example fails.
  - Each lesson's steps SHALL be completed and checked off by an end-to-end test in the practice project, which also regenerates the lesson screenshots.

## 13. Security posture

- **NFR-20**: In solo mode, agents MAY run arbitrary code on the development machine, and container or VM sandboxing is not required. *(Rev 14:)* On a multi-user Linux server, every user's agent work SHALL run in that user's workspace (FR-207).
- **NFR-21**: Access control relies on network exposure and Tailscale (§4.2) in solo mode, and on sign-in and per-project permissions (§4.4) in multi-user mode. The application SHALL NOT expose itself on external interfaces without the challenge required by FR-18.
- **NFR-23**: *(rev 13)* Before every release, the **harness check** (`scripts/harness_e2e.sh`) SHALL pass for each harness marked verified. It SHALL run in its own data folder and herdr session, prove that a member works a task without stopping, that a one-question helper answers, that answering a blocking question resumes the task, and that a member takes a second task in its conversation, and SHALL print the screen of any harness that stopped.
- **NFR-24**: *(rev 13)* With warm harnesses (FR-204), a user's Assistant SHALL begin replying within 5 seconds, and a member SHALL begin a task within 10 seconds of it being started.
- **NFR-25**: *(rev 14)* A stopped workspace SHOULD start in under 3 seconds as a container and under 5 seconds as a microVM, and a running workspace SHOULD add no more than 10% to a task's time compared with running on the host.
- **NFR-22**: *(rev 12)* Passwords, tokens and OIDC secrets SHALL never be written to logs, the history or the repository. Tokens SHALL be stored hashed. Security-relevant settings SHALL be changeable only by an Owner or Admin, and each change SHALL be recorded.

## 14. Open questions

| # | Question |
|---|---|
| Q1 | ~~For access through an external network interface (FR-18), what should the challenge be?~~ **Answered 2026-11-12:** sign-in (§4.4): local accounts with a second step first, OIDC required next (provider not chosen yet). |
| Q2 | Check that each target harness supports headless operation **and** reports context-window usage. Claude Code reports usage in its streaming JSON output. OpenCode offers a non-interactive mode (`opencode run`) and a server mode; how it reports context usage needs to be confirmed. Antigravity's headless and CLI support needs to be confirmed. |
| Q3 | When smart handoffs graduate from experimental, what is the default context threshold, and should it be configurable per harness or model? What evidence (rework rate, handoff quality vs. compaction) decides that they are ready? |
| Q4 | ~~Can multiple humans collaborate on the same projects?~~ **Answered 2026-11-11/12:** yes, on one shared server, with their own accounts, statuses and per-project access roles (§4.4). |
| Q5 | What is the exact default git workflow (branch naming, PR/MR requirements, merge strategy)? |
| Q6 | ~~What is the mechanism and channel for auto-updates?~~ **Answered:** GitHub Releases with signed checksums (ADR-004). Rev 4 adds the in-app change log and guided updates with step progress (ADR-011). |
| Q7 | ~~How should attachable (headed) sessions work?~~ **Answered (rev 3):** every session runs in herdr; see ADR-009. |
| Q8 | ~~Conversation rotation policy (FR-76)?~~ **Answered 2026-11-08:** keep one conversation per member across tasks until context reaches the handoff threshold (75%), then rotate, starting from a handoff plus the member's notes. Lens starts a fresh conversation for every review. See ADR-010. |
| Q9 | ~~Windows support level?~~ **Answered 2026-11-08:** ship Windows at herdr's beta level, with herdr in the Windows CI job. A herdr capability missing on Windows is shown as unavailable there. See ADR-009. |
| Q10 | ~~Archetype change propagation (FR-12b)?~~ **Answered 2026-11-08:** a proposal per affected member, plus one action to apply the change to every member that has not overridden the changed fields. See ADR-010. |
| Q11 | ~~Should the doctor's prompt tier (FR-07b) ever run on its own?~~ **Answered 2026-11-09:** no. It runs only when asked for, because it uses tokens. See ADR-012. |
| Q12 | ~~Should lessons use a sandbox or a real project?~~ **Answered 2026-11-09:** a practice project with the scripted fake harness by default (FR-136); the last lesson moves to a real project. Claude Code is the main harness example (FR-137). See ADR-012. |
| Q13 | ~~Which user management take?~~ **Answered 2026-11-20:** Take 1, the admin console. Dev mode first (§4.5), then the personal Assistant (§4.6). Project-defined roles (FR-176) are not planned. |
| Q14 | Which OIDC provider will be used first (FR-162)? |
| Q15 | Is the second sign-in step required for everyone, or only for admins and approvers (FR-161)? May project admins invite people who have no account yet? |
| Q16 | May an AI member ever approve a gate (for example the AI reviewer on a low-risk gate), or never (FR-179)? *(Rev 13: the personal Assistant never decides on its own; it drafts for its user, FR-198.)* |
| Q17 | How much memory may warm harnesses use by default (FR-204)? To be set from measurements in dev mode. |
| Q18 | Will agents build untrusted code from external repositories (which argues for strong isolation, FR-208, by default)? |
| Q19 | Do projects need to bring their own Dockerfiles, or is a registry image enough (FR-209)? Building images needs a builder (for example BuildKit) on the server. |
| Q20 | Will the server be bare-metal Linux, or a cloud VM (microVMs need KVM, which many cloud VMs don't offer)? |

## 15. Out of scope (for now)

- Email or other external notifications
- Container or VM sandboxing of agents in solo mode (on a multi-user Linux server, see §8.6)
- Running agent work on people's own machines for a shared server (local runners), and syncing a local go-build-it with a server (decided 2026-11-20: the server holds the state and does the setup, §4.7)
- Cost and budget tracking beyond detecting subscription limits
- Consuming go-build-it as a Go module
- Agent process hosts other than herdr (tmux, a built-in PTY, headless `-p` / `stream-json` runs)
- Driving herdr on remote machines (herdr's saved SSH machines); go-build-it uses the local gbi session only
- Parsing harness dialogs from the screen to answer them on the user's behalf, except the folder trust prompt for folders go-build-it created (FR-07f)
