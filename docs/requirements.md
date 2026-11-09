# go-build-it — Requirements

Status: Draft · Last updated: 2026-11-08 (rev. 3: herdr as the only agent backend; per-project team members from global archetypes. See [ADR-009](decisions/ADR-009-herdr-backend.md) and [ADR-010](decisions/ADR-010-project-team-members.md))

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
- **FR-08**: The background service (FR-02) SHALL ensure both go-build-it and its herdr session are running at login. Stopping or updating go-build-it SHALL NOT stop running harnesses, because herdr owns them.
- **FR-02**: The application SHALL be installable globally on a machine and SHOULD run as a background service (launchd on macOS, systemd on Linux, a Windows service on Windows).
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
- **FR-13**: Archetypes and project members' tuning SHALL be edited in the artifact editor (FR-45a). The user can write the definition directly, or chat with an AI agent that helps develop it (e.g. interviews the user, suggests instructions, checks for conflicts with global standards).

### 4.2 Access and identity

Authentication is delegated to Tailscale. go-build-it does not run its own login system for local or tailnet access.

- **FR-15**: The application SHALL detect how each request reaches it and classify it as one of:
  1. **Local**: a loopback connection (`localhost` / `127.0.0.1` / `::1`).
  2. **Tailscale Serve**: proxied by `tailscale serve` on the tailnet.
  3. **External interface**: a direct connection on a non-loopback network interface.
- **FR-16**: **Local** requests SHALL NOT be challenged. They SHALL be attributed to a configurable default local user.
- **FR-17**: **Tailscale Serve** requests SHALL be treated as trusted, the same as local requests, because the user has already authenticated with Tailscale. When Tailscale identity headers are present (e.g. `Tailscale-User-Login`), the request SHOULD be attributed to that Tailscale user, so actions in the history show who performed them.
- **FR-18**: **External interface** requests SHALL be challenged before access is granted. See open question Q1.
- **FR-19**: By default the application SHALL listen only on loopback. Listening on external interfaces SHALL require explicit configuration.
- **FR-20**: The application SHALL help the user set up Tailscale Serve, for example by detecting whether Tailscale is installed and logged in and then offering to configure `tailscale serve`. It SHALL show clear guidance when Tailscale is missing.

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

- **FR-26**: Users SHOULD be able to define additional custom roles.

## 5. User interfaces

- **FR-30**: Everything that can be done in the web UI SHALL also be possible through the CLI, so AI agents can operate the application. (AI-native requirement.)
- **FR-31**: The application SHALL be API-first. The web UIs and the CLI SHALL be clients of the same HTTP API. The API SHOULD be described by an OpenAPI spec.
- **FR-32**: CLI commands SHALL support machine-readable output (`--json`).
- **FR-33**: The application SHOULD expose an MCP server so external agents can drive it.
- **FR-34**: The application SHALL provide a **desktop web UI** designed for laptop and desktop screens.
- **FR-35**: The application SHALL provide a separate **mobile web UI** designed for phones. It SHALL put the most common mobile tasks first: answering questions, reviewing and giving feedback, checking project status, and approving review gates.
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

## 7. Global standards

- **FR-50**: The application SHALL maintain a set of global standards that apply to every project. Examples:
  - standard backend stack
  - standard frontend stack
  - unit tests required
  - end-to-end tests required
  - documentation requirements
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
- **FR-66**: The live view of an AI team member SHALL offer two modes:
  - a **structured view**: events and messages read from the harness's transcript (edits, commands, test results, messages)
  - a **terminal view**: the member's herdr pane streamed live to the desktop and mobile UI (herdr `terminal session observe`), with the option to take control (`terminal session control`) and type, resize and use keys. Several viewers can watch at once; one controls at a time.
- **FR-67**: The gbi herdr session SHALL be organized so that opening it shows everything go-build-it is doing:
  - one herdr workspace per project, with each AI team member's worktree grouped under it
  - each member's pane labelled with its name, role and current task, plus small status values such as context usage (herdr pane metadata)
  - each project workspace showing a summary such as "2 working · 1 needs you" (herdr workspace metadata)
  - review gates and questions that need the human raised as herdr notifications
- **FR-64**: Each AI team member SHALL work in its own long-lived git worktree, so concurrent members don't interfere. Each task SHALL get a fresh branch in that worktree. The application SHALL keep creating worktrees itself and SHALL ask herdr to open them grouped under the project workspace.
- **FR-68**: The application SHALL treat herdr's agent state as the source of truth for whether a member is working, blocked, done or idle. A **blocked** member SHALL appear in the UI as needing the human, linked to its live terminal view. The application SHALL NOT try to understand harness dialogs from the screen. It SHOULD avoid permission prompts through permission modes and per-project allowlists, and MAY use harness hooks (e.g. Claude Code's permission hooks) to offer structured approvals.
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
  - an assessment against the global and project standards
  - an assessment against the feedback given during mockup reviews and dress rehearsals: addressed or not
  - an assessment against the requirements in scope: met, partly met or not met, with evidence
  - a link to the pull or merge request, where the code diff remains available

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

## 13. Security posture

- **NFR-20**: Agents MAY run arbitrary code on the development machine. Container or VM sandboxing is explicitly **not** required.
- **NFR-21**: Access control relies on network exposure and Tailscale (§4.2). The application SHALL NOT expose itself on external interfaces without the challenge required by FR-18.

## 14. Open questions

| # | Question |
|---|---|
| Q1 | For access through an external network interface (FR-18), what should the challenge be: a local password or token, an OAuth/OIDC provider, or refusing external access completely and requiring Tailscale Serve? |
| Q2 | Check that each target harness supports headless operation **and** reports context-window usage. Claude Code reports usage in its streaming JSON output. OpenCode offers a non-interactive mode (`opencode run`) and a server mode; how it reports context usage needs to be confirmed. Antigravity's headless and CLI support needs to be confirmed. |
| Q3 | When smart handoffs graduate from experimental, what is the default context threshold, and should it be configurable per harness or model? What evidence (rework rate, handoff quality vs. compaction) decides that they are ready? |
| Q4 | Can multiple humans collaborate on the same projects (e.g. different Tailscale users with different roles), or is it single-user with optional reviewers? |
| Q5 | What is the exact default git workflow (branch naming, PR/MR requirements, merge strategy)? |
| Q6 | What is the mechanism and channel for auto-updates (GitHub Releases, Homebrew tap, etc.)? |
| Q7 | ~~How should attachable (headed) sessions work?~~ **Answered (rev 3):** every session runs in herdr; see ADR-009. |
| Q8 | ~~Conversation rotation policy (FR-76)?~~ **Answered 2026-11-08:** keep one conversation per member across tasks until context reaches the handoff threshold (75%), then rotate, starting from a handoff plus the member's notes. Lens starts a fresh conversation for every review. See ADR-010. |
| Q9 | ~~Windows support level?~~ **Answered 2026-11-08:** ship Windows at herdr's beta level, with herdr in the Windows CI job. A herdr capability missing on Windows is shown as unavailable there. See ADR-009. |
| Q10 | ~~Archetype change propagation (FR-12b)?~~ **Answered 2026-11-08:** a proposal per affected member, plus one action to apply the change to every member that has not overridden the changed fields. See ADR-010. |

## 15. Out of scope (for now)

- Email or other external notifications
- Container or VM sandboxing of agents
- Cost and budget tracking beyond detecting subscription limits
- Consuming go-build-it as a Go module
- Agent process hosts other than herdr (tmux, a built-in PTY, headless `-p` / `stream-json` runs)
- Driving herdr on remote machines (herdr's saved SSH machines); go-build-it uses the local gbi session only
- Parsing harness dialogs from the screen to answer them on the user's behalf
