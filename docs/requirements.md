# go-build-it — Requirements

Status: Draft · Last updated: 2026-11-11 (rev. 8: docs/gbi.toml controls the Artifacts sub-tabs, which ones exist, their order and their names, including your own roles such as vision, FR-151. Rev. 7: project documents with HTML entry points, explicit labels, a viewer that follows links, mockup-to-app comparison and traceability, FR-140–FR-150, see [ADR-014](decisions/ADR-014-project-documents.md). Rev. 6: go-build-it answers a harness's folder trust prompt for folders it created, FR-07f, see [ADR-013](decisions/ADR-013-folder-trust.md). Rev. 5: a doctor that proves gbi can start herdr and a harness and explains failures to people and AI agents, FR-07a–FR-07e; install by giving an AI agent the GitHub URL, FR-02a–FR-02b; lessons, a practice project, harness control and design rationale, FR-134–FR-139; see [ADR-012](decisions/ADR-012-doctor-and-learning.md). Rev. 4: in-app change log and guided updates, FR-06a–FR-06e, see [ADR-011](decisions/ADR-011-changelog-guided-updates.md). Rev. 3: herdr as the only agent backend; per-project team members from global archetypes, see [ADR-009](decisions/ADR-009-herdr-backend.md) and [ADR-010](decisions/ADR-010-project-team-members.md))

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
| Q6 | ~~What is the mechanism and channel for auto-updates?~~ **Answered:** GitHub Releases with signed checksums (ADR-004). Rev 4 adds the in-app change log and guided updates with step progress (ADR-011). |
| Q7 | ~~How should attachable (headed) sessions work?~~ **Answered (rev 3):** every session runs in herdr; see ADR-009. |
| Q8 | ~~Conversation rotation policy (FR-76)?~~ **Answered 2026-11-08:** keep one conversation per member across tasks until context reaches the handoff threshold (75%), then rotate, starting from a handoff plus the member's notes. Lens starts a fresh conversation for every review. See ADR-010. |
| Q9 | ~~Windows support level?~~ **Answered 2026-11-08:** ship Windows at herdr's beta level, with herdr in the Windows CI job. A herdr capability missing on Windows is shown as unavailable there. See ADR-009. |
| Q10 | ~~Archetype change propagation (FR-12b)?~~ **Answered 2026-11-08:** a proposal per affected member, plus one action to apply the change to every member that has not overridden the changed fields. See ADR-010. |
| Q11 | ~~Should the doctor's prompt tier (FR-07b) ever run on its own?~~ **Answered 2026-11-09:** no. It runs only when asked for, because it uses tokens. See ADR-012. |
| Q12 | ~~Should lessons use a sandbox or a real project?~~ **Answered 2026-11-09:** a practice project with the scripted fake harness by default (FR-136); the last lesson moves to a real project. Claude Code is the main harness example (FR-137). See ADR-012. |

## 15. Out of scope (for now)

- Email or other external notifications
- Container or VM sandboxing of agents
- Cost and budget tracking beyond detecting subscription limits
- Consuming go-build-it as a Go module
- Agent process hosts other than herdr (tmux, a built-in PTY, headless `-p` / `stream-json` runs)
- Driving herdr on remote machines (herdr's saved SSH machines); go-build-it uses the local gbi session only
- Parsing harness dialogs from the screen to answer them on the user's behalf
