# go-build-it — Requirements

Status: Draft · Last updated: 2026-11-07

## 1. Overview

go-build-it is a Go application with an embedded React frontend that manages long-running software project development. It is installed globally on a machine and provides a web UI and a CLI. A team of AI personas and humans works through it to take projects from problem definition to released software. The app coordinates external agent harnesses such as Claude Code, Antigravity and OpenCode, and it drives them headlessly.

### Conventions

- **SHALL**: required for the release the requirement is assigned to.
- **SHOULD**: strongly desired. It can be deferred, but the reason must be recorded.
- **MAY**: optional.
- Requirement IDs (`FR-xx`, `NFR-xx`) are stable. Tests, the UAT walkthrough and the docs refer to them by ID.

## 2. Glossary

| Term | Definition |
|---|---|
| **Project** | A software product managed by go-build-it. It is backed by a git repository and has its own requirements, values, roadmap, task queue and history. |
| **Team** | A user-defined set of team members assigned to work on one or more projects. |
| **Team member** | Either an **AI persona** or a **human user**, holding one role. |
| **AI persona** | A configured agent identity: role, instructions or prompt, assigned harness, model, and any role-specific standards. |
| **Role** | A responsibility within the team. See §4.3. |
| **Agent harness** | An external agent CLI (e.g. Claude Code, Antigravity, OpenCode, Pi-agent) that go-build-it drives headlessly to carry out a task. |
| **Session** | One run of a harness working on a task, with its own context window. |
| **Handoff** | A curated briefing produced when a session ends, mainly when its context window is nearly full. The next session starts from it instead of from a raw memory dump. |
| **Global standards** | Rules and conventions that apply to every project, e.g. the standard backend stack, frontend stack and test requirements. |
| **Values** | A project's guiding principles, discovered as work proceeds. They are used to judge and generate solutions. |
| **Question** | A request for input sent from a team member (usually an AI persona) to a human. It carries the context and artifacts needed to answer it. |
| **Review gate** | A planned point in a project's roadmap where work pauses for human review, such as a dress rehearsal. |
| **Dress rehearsal** | A guided, feature-by-feature walkthrough of the application, done as a tabletop exercise before implementation or as a UAT run after it. The human reviewer gives feedback on each feature. |
| **Option set** | Two or three alternative approaches to a problem, presented for the human to choose among or combine. |

## 3. Platform, installation and runtime

- **FR-01**: The application SHALL be a single Go binary with the React frontend embedded.
- **FR-02**: The application SHALL be installable globally on a machine and SHOULD run as a background service (launchd on macOS, systemd on Linux, a Windows service on Windows).
- **NFR-01**: macOS SHALL be the initial target platform. Linux and Windows SHALL be supported later. Code SHALL NOT assume macOS-only paths, shells or tools without a cross-platform abstraction.
- **FR-03**: All application state SHALL live under `~/.go-build-it/` (`%USERPROFILE%\.go-build-it\` on Windows). This includes:
  - configuration files
  - the SQLite database
  - logs and session transcripts
- **FR-04**: Persistence SHALL use SQLite.
- **FR-05**: The application SHALL use calendar versioning in the format `YYYY.MM.DD.N`, where `N` is the release number for that day, starting at 1. The application is not intended to be consumed as a Go module.
- **FR-06**: The application SHALL be able to install new releases automatically:
  - Releases SHALL be verified by checksum and SHOULD be signed.
  - An update SHALL NOT be applied while agent sessions are running. It SHALL wait for them to finish or for an idle window.
  - The previous version SHALL be kept so the user can roll back.
  - The user SHALL be able to turn automatic updates off.

## 4. Users, team and roles

### 4.1 Team members

- **FR-10**: A team member SHALL be either an AI persona or a human user.
- **FR-11**: The user SHALL be able to create, edit and remove teams, and assign teams to projects.
- **FR-12**: Each AI persona SHALL be configurable with: role, instructions, assigned harness, model, and any role-specific standards that add to the global standards.

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
- **FR-46**: Project artifacts SHOULD be stored in the project's git repository (e.g. under `docs/`) so they are versioned with the code. Operational state (queues, sessions, history) SHALL live in the application database.

## 7. Global standards

- **FR-50**: The application SHALL maintain a set of global standards that apply to every project. Examples:
  - standard backend stack
  - standard frontend stack
  - unit tests required
  - end-to-end tests required
  - documentation requirements
- **FR-51**: Standards SHALL be authored as Markdown and managed in the UI and the CLI.
- **FR-52**: A project MAY override or extend a global standard. Every override SHALL be recorded with its reason.
- **FR-53**: Standards SHALL be supplied to every agent session automatically, for example by generating the harness's instruction file (`CLAUDE.md`, `AGENTS.md` or equivalent), or by injecting them into the session context.
- **FR-54**: The Reviewer role SHALL check completed work against the applicable standards.

## 8. Agent harness orchestration

### 8.1 Harness integration

- **FR-60**: The application SHALL drive agent harnesses headlessly through their command-line interfaces, at the direction of the development team.
- **FR-61**: Harness support SHALL be implemented through an adapter interface providing:
  - start a session with a prepared context
  - stream output and events
  - send input
  - interrupt or cancel
  - report context-window usage
  - report exit status
- **FR-62**: Initial adapters: **Claude Code** (Anthropic subscription), **Antigravity** (Google / Gemini subscription) and **OpenCode** (open-source harness that works with multiple model providers). Pi-agent and others SHOULD be addable through the same interface.
- **FR-63**: Harnesses and their settings (binary path, model, flags, permission mode) SHALL be configurable in the UI and the CLI.
- **FR-64**: Concurrent sessions on the same project SHOULD work in isolated workspaces, such as git worktrees, so they don't interfere with each other.
- **FR-65**: Cost is limited by the user's AI subscriptions, so no budget controls are needed. The application SHALL detect subscription rate or usage limits reported by a harness. It SHALL then pause the affected work and resume it automatically when the limit resets.

### 8.2 Context management and smart handoffs

- **FR-70**: The application SHALL prepare each session's starting context deliberately: the task, the relevant requirements, standards, values, and the incoming handoff (if any). It SHALL NOT dump all of the project's history.
- **FR-71**: The application SHALL monitor each session's context-window usage while it runs.
- **FR-72**: When usage reaches a configurable threshold (e.g. 70–80%), the application SHALL step in and start a smart handoff **before** the harness auto-compacts.
- **FR-73**: A handoff SHALL be a curated briefing, written as if handing responsibility to another person. At minimum it SHALL contain:
  - objective and definition of done
  - current state and what has been completed
  - decisions made and why
  - open problems, risks and dead ends to avoid
  - specific next steps
  - pointers to the relevant files and artifacts (references, not copies)
- **FR-74**: The next session SHALL start from the handoff and FR-70's prepared context. Every handoff SHALL be stored in the history.
- **FR-75**: Tasks SHOULD be sized so most of them fit in one session. The Project Manager role SHOULD split tasks that repeatedly need handoffs.

### 8.3 Task queues

- **FR-80**: Each project SHALL have a persistent task queue that can manage large numbers of tasks.
- **FR-81**: Queues SHALL support:
  - priorities
  - dependencies between tasks
  - concurrency limits per project and globally
  - retries
  - assignment of tasks to roles or personas
- **FR-82**: Queue state SHALL survive application and machine restarts. Interrupted sessions SHALL be resumed or restarted from their latest handoff.

## 9. Git, GitHub and GitLab workflow

- **FR-90**: The application SHALL work with both GitHub (`gh`) and GitLab (`glab`), and SHALL detect which one each project uses and whether its CLI is authenticated.
- **FR-91**: The application SHALL enforce a defined development workflow, configurable per project with a global default. For example: branch per task → commits → push → PR/MR → review → merge.
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
- **FR-103**: Questions MAY offer structured answers (choices, yes/no, pick-an-option) as well as free text.
- **FR-104**: A **Questions dashboard** (on desktop and mobile) SHALL list open questions across all projects. It SHALL put blocking questions first and SHALL let the user answer each question in place, with its context, so they can get through them quickly.
- **FR-105**: Answers SHALL be sent back to the asking session or task automatically. Answers that reveal a project value SHALL be offered for recording in the project's values (§10.4).
- **FR-106**: Email and other out-of-app notifications are out of scope for now. In-app live updates (FR-36) are in scope.

### 10.2 Feedback on completed work

- **FR-110**: The application SHALL provide an easy way to give feedback on completed work. The verdicts are: approve, request changes, or reject. Comments SHALL attach to a specific artifact, screen or diff location.
- **FR-111**: Feedback SHALL go into the context of the follow-up task automatically, and SHALL be recorded in the history.

### 10.3 Review gates and dress rehearsals

- **FR-115**: Roadmaps SHALL support **review gates**: defined points where work pauses for human review and feedback before going on.
- **FR-116**: A review gate MAY be a **dress rehearsal**. The application walks the human step by step through every feature or scenario in scope. It can do this as a tabletop exercise before implementation (using mockups and scenarios) or as a UAT run after it (using the working application).
- **FR-117**: During a dress rehearsal, the human SHALL be able to give feedback on each step, and SHALL be able to talk with the agents live to request changes or improvements as the rehearsal goes on.
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
- **NFR-11**: It SHALL always be possible to determine what the system is currently doing and why. Live views SHALL show active sessions, their tasks, and their context usage.
- **NFR-12**: Records SHALL be structured so AI agents can analyze the application's own performance. Examples:
  - handoff quality
  - rework rates
  - question volume
  - time per task
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
| Q3 | What is the default context threshold for a handoff, and should it be configurable per harness or model? |
| Q4 | Can multiple humans collaborate on the same projects (e.g. different Tailscale users with different roles), or is it single-user with optional reviewers? |
| Q5 | What is the exact default git workflow (branch naming, PR/MR requirements, merge strategy)? |
| Q6 | What is the mechanism and channel for auto-updates (GitHub Releases, Homebrew tap, etc.)? |

## 15. Out of scope (for now)

- Email or other external notifications
- Container or VM sandboxing of agents
- Cost and budget tracking beyond detecting subscription limits
- Consuming go-build-it as a Go module
