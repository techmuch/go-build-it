"""Add the rev 12–13 phases to the tracker: P26 (harnesses that work in herdr, built while making
Last Card) and P27–P33 (dev mode, users and access through the admin console, the personal
Assistant and warm harnesses, sign-in, credentials, sign-off, OIDC).

Idempotent: re-running replaces P26–P33 tasks but keeps any status already set on them. Re-parses
docs/requirements.md and fails if a requirement is uncovered or a dependency is unknown.

Usage: python3 scripts/docs/add_rev13_phases.py
"""
import datetime, json, os, re, sys

ROOT = os.path.join(os.path.dirname(__file__), "..", "..")
REQ_MD = os.path.join(ROOT, "docs", "requirements.md")
DATA = os.path.join(ROOT, "docs", "tracker", "tracker-data.js")

src = open(DATA).read()
head = src[: src.index("window.TRACKER")]
D = json.loads(src[src.index("{"): src.rindex("}") + 1])

md = open(REQ_MD).read()
reqs, questions, section = {}, {}, ""
for line in md.splitlines():
    h = re.match(r"^#{2,3} (.+)", line)
    if h:
        section = h.group(1)
    m = re.match(r"^- \*\*((?:FR|NFR)-\d+[a-z]?)\*\*: (.*)", line)
    if m:
        text = re.sub(r"\*\*|\*|`", "", m.group(2))
        first = re.split(r"(?<=[.:])\s", text, maxsplit=1)[0]
        reqs[m.group(1)] = {"text": first[:180], "section": section}
    q = re.match(r"^\| (Q\d+) \| (.*) \|$", line)
    if q:
        questions[q.group(1)] = re.sub(r"\*\*|`|~~", "", q.group(2))

PHASES = [
    dict(id="P26", name="Harnesses that work in herdr", mvp=False,
         goal="Real AI harnesses run go-build-it's work in herdr without silent stalls: prompts are delivered, helpers answer, answered questions resume work, reviews see the whole change, and a check proves it before every release.",
         gate="Harness check",
         gateCriteria=["scripts/harness_e2e.sh passes for Claude Code: a task without a permission stop, a helper answer, a blocking question that resumes by itself, a second task in the same conversation",
                       "A stopped harness says what its screen shows (for example a model or plan menu)",
                       "Approving a review with evidence gaps needs a reason; standards that don't apply aren't counted"]),
    dict(id="P27", name="Dev mode and users", mvp=False,
         goal="One person on one machine can play two users and try multi-user work: users exist, access roles and grants decide what each may do, every route is checked, and the dev bar switches users and explains each allow or deny.",
         gate="Two users in dev mode",
         gateCriteria=["gbi serve --dev creates david (Owner) and maria (Contributor); the dev bar switches between them on every desktop and phone screen",
                       "As maria, approving a gate is denied and the bar names the missing permission; as david it is allowed",
                       "Every action is recorded as the user acted as, marked dev with who switched",
                       "Dev mode refuses to start while Tailscale Serve or external interfaces are on"]),
    dict(id="P28", name="The personal Assistant and warm harnesses", mvp=False,
         goal="Every user has an Assistant, made from the Assistant archetype and tuned in their profile, always running while they are signed in, shown on every screen, and the one every side chat talks to; harnesses are kept warm so replies start quickly.",
         gate="Two Assistants at work",
         gateCriteria=["Switching to maria starts her Assistant; both Assistants show their status in the top bar and on the phone",
                       "Every side chat (artifacts, archetypes, standards, Learn) talks to your Assistant, which drafts a change you accept",
                       "The Assistant drafts an answer and a review verdict for its user and cannot submit either itself",
                       "A warm Assistant starts replying within 5 s; a warm member starts a task within 10 s (NFR-24)"]),
    dict(id="P29", name="Sign-in", mvp=False,
         goal="Multi-user mode: people sign in with local accounts and a second step; the CLI, MCP and agent sessions use expiring tokens; loopback and Tailscale are trusted only as configured.",
         gate="Sign-in review",
         gateCriteria=["The owner switches to multi-user mode; maria accepts an invitation, sets a password and a passkey, and signs in",
                       "gbi login gets a token; an agent session's token ends with its session; X-GBI-Persona is ignored",
                       "Loopback is refused until trusted as a named service account; an unlinked Tailscale login gets the sign-in page"]),
    dict(id="P30", name="People & access (the admin console)", mvp=False,
         goal="Take 1: People & access with users, statuses, groups, service accounts and custom roles; each project has an Access tab that answers \"can this person do this, and why?\"; access is requested and granted from the inbox.",
         gate="Access review by the owner",
         gateCriteria=["Suspending a user ends their sessions and tokens at once and is recorded with the reason",
                       "A custom role made by ticking permissions shows who it affects before saving",
                       "Check access explains an allow or a deny for any user, AI member and permission",
                       "A person without access requests it; the request lands in the project admin's inbox and can be granted"]),
    dict(id="P31", name="Harness credentials", mvp=False,
         goal="Shared or Per user credentials, as one setting: harnesses signed in before go-build-it starts, or API keys per harness and model, each tested by the doctor; work runs on the right person's credentials.",
         gate="Credentials review",
         gateCriteria=["In Per user mode, maria signs in her own Claude Code from her profile and her Assistant runs on it",
                       "An API key is saved, tested, never shown again and never in a log",
                       "Automatic dispatch starts paid work only for a project that names whose credentials pay"]),
    dict(id="P32", name="Sign-off routes and access review", mvp=False,
         goal="Gates are signed off by the groups that must sign them, in order or together; sign-offs are tied to versions; an access review finds risky access and exports it.",
         gate="Sign-off rehearsal",
         gateCriteria=["A gate with a two-step route reaches each signer's inbox and shows progress; changing the artifact withdraws the sign-off",
                       "Nobody can sign off their own work",
                       "An access review lists unused accounts and self-approval risks and exports CSV"]),
    dict(id="P33", name="Single sign-on (OIDC)", mvp=False,
         goal="People sign in with the organisation's identity provider; groups can follow its claims; a break-glass account stays.",
         gate="SSO sign-in",
         gateCriteria=["Settings test the OIDC connection and show the redirect URL",
                       "A user signs in with SSO and is matched by verified email; a group follows a claim",
                       "Local accounts can be turned off with one break-glass account kept"]),
]
T = []
def t(id, title, reqs_, views, size, deps, q=None, done="", status="todo"):
    T.append(dict(id=id, phase=id.split("-")[0], title=title, reqs=reqs_, views=views, size=size,
                  deps=deps, status=status, done=done, q=q or []))
D_ = "done"

# P26: built and proven while making Last Card with go-build-it (2026-11-15 to 2026-11-20).
t("P26-01", "Resend a task a starting harness dropped (members)", ["FR-60", "FR-68"], [], "S", [], done="internal/sessions/run.go", status=D_)
t("P26-02", "Claude Code members run in auto permission mode; a mode in the harness flags replaces it", ["FR-63"], [], "S", [], done="internal/harness/profile.go", status=D_)
t("P26-03", "Answering the last blocking question resumes the task, waiting for a busy member; requested changes start the follow-up", ["FR-105", "FR-111"], [], "S", [], done="internal/questions, internal/app/app.go resumeTask", status=D_)
t("P26-04", "One-question helpers: folder trust answered before the prompt, dropped prompts resent", ["FR-206", "FR-07f"], [], "S", [], done="internal/sessions/ask.go", status=D_)
t("P26-05", "Reviewer gets the task, the whole change and the worktree; standards can be marked does not apply; npm and Node tests counted", ["FR-112", "FR-112a"], [], "M", [], done="internal/reviews", status=D_)
t("P26-06", "Evidence gaps: no verdict preselected; approving with gaps needs a reason (desktop and phone); not run is never pass", ["FR-112a", "FR-110"], ["A-v09", "A-M5"], "S", ["P26-05"], done="Review.tsx, Mobile.tsx, reviews.go", status=D_)
t("P26-07", "A stopped harness's screen becomes the reason the person sees", ["FR-205", "FR-68"], [], "S", [], done="internal/sessions/trust.go ScreenSummary", status=D_)
t("P26-08", "Partial settings changes merge at every depth", ["FR-63"], [], "S", [], done="internal/app/routes.go mergeJSON", status=D_)
t("P26-09", "One document action is one commit", ["FR-46"], [], "S", [], done="courier.CommitArtifacts", status=D_)
t("P26-10", "Harness check script (NFR-23); measured on Claude Code with model opus", ["NFR-23", "FR-07b"], [], "M", ["P26-01", "P26-03", "P26-04", "P26-07"], done="scripts/harness_e2e.sh", status=D_)
t("P26-11", "Screens as built with a review against the vision; design mockups archived", ["FR-141", "FR-133"], [], "M", [], done="docs/mockups/index.html, scripts/e2e/capture_screens.cjs", status=D_)
t("P26-12", "Requirements rev 12–13, ADR-016, user management mockups (Take 1 chosen, section D)", ["FR-160", "FR-190", "FR-194", "FR-200"], ["UM-1a", "UM-D1", "UM-D2", "UM-D3", "UM-D4"], "M", [], done="docs/requirements.md rev 13; docs/decisions/ADR-016", status=D_)
t("P26-13", "Doctor names a herdr socket path that is too long for the OS", ["FR-07b", "FR-07"], [], "S", ["P26-10"])
t("P26-14", "Release 2026.11.20.1: signed binaries, install check, harness check", ["FR-05", "FR-06", "NFR-23"], [], "S", ["P26-10", "P26-11", "P26-12"])

# P27: dev mode and users, before any sign-in.
t("P27-01", "Users table and store: ID, name, email, status, created and last seen; history points at user IDs", ["FR-160", "FR-169"], ["UM-1a"], "M", ["P26-14"])
t("P27-02", "Permissions, built-in access roles and grants per project (users, groups later, AI members); project default and private projects", ["FR-172", "FR-173", "FR-177"], ["UM-1b", "UM-1c"], "L", ["P27-01"])
t("P27-03", "Every route declares its permission in the route table; one check for UI, CLI and MCP; denials name the missing permission", ["FR-185", "FR-30", "FR-31"], [], "L", ["P27-02"])
t("P27-04", "AI members checked the same way, with their archetype's ceiling; acting for a person gets both their permissions", ["FR-179"], [], "M", ["P27-03"])
t("P27-05", "Solo and multi-user modes; at least one Owner always", ["FR-163", "FR-188"], ["UM-S2"], "S", ["P27-01"])
t("P27-06", "Dev mode: gbi serve --dev, refused while reachable from other machines, shown in doctor and Settings", ["FR-190"], ["UM-D1"], "S", ["P27-05"])
t("P27-07", "Dev bar on every desktop and phone screen; switch user in one click and gbi dev as USER", ["FR-191"], ["UM-D1"], "M", ["P27-06", "P27-03"])
t("P27-08", "Two dev users on first start; add more from the bar", ["FR-192"], ["UM-D1"], "S", ["P27-07"])
t("P27-09", "Dev actions recorded as the user, marked dev with who switched; the bar explains the last allow or deny", ["FR-193", "FR-186", "FR-178"], ["UM-D1"], "M", ["P27-07"])
t("P27-10", "Per-user inbox, projects and pins follow the acting user", ["FR-170"], ["UM-1d"], "M", ["P27-07"])
t("P27-11", "Tests (route permission table, dev bar UAT for two users), docs, CHANGELOG, release", ["FR-133", "FR-130", "NFR-23"], ["UM-D1"], "M", ["P27-08", "P27-09", "P27-10", "P27-04"])

# P28: the personal Assistant and warm harnesses.
t("P28-01", "Assistant archetype (system-wide); each user's Assistant made from it; tuning and harness/model choice in the profile", ["FR-195", "FR-12b"], ["UM-D3"], "M", ["P27-11"])
t("P28-02", "Assistant always running while its user is signed in or switched to; own herdr pane per user; idle stop; conversation resumed", ["FR-196"], ["UM-D3"], "M", ["P28-01"])
t("P28-03", "Assistant status chip on every desktop and phone screen; the Assistant panel", ["FR-197"], ["UM-D2"], "M", ["P28-02"])
t("P28-04", "Every side chat talks to your Assistant (artifacts, archetypes, standards, persona coach, Learn)", ["FR-194"], ["UM-D2"], "L", ["P28-03"])
t("P28-05", "Assistant acts with its user's permissions and a token bound to both; drafts decisions for the user to confirm", ["FR-198", "FR-199", "FR-167"], ["UM-D2"], "M", ["P28-02", "P27-04"])
t("P28-06", "Warm harnesses: Assistants and members of projects with queued work kept started within count and memory limits; Settings shows each with memory", ["FR-204"], ["UM-D4"], "L", ["P28-02"], q=["Q17"])
t("P28-07", "Measure start and reply times and memory per harness; set the defaults; NFR-24 in the harness check", ["NFR-24", "NFR-23"], ["UM-D4"], "S", ["P28-06"], q=["Q17"])
t("P28-08", "Tests, docs, CHANGELOG, release", ["FR-133", "FR-130"], ["UM-D2"], "M", ["P28-04", "P28-05", "P28-07"])

# P29: sign-in.
t("P29-01", "Local accounts: Argon2id passwords, leaked-password check, admin reset", ["FR-161", "NFR-22"], ["UM-S1"], "M", ["P27-11"])
t("P29-02", "Second step: passkeys (WebAuthn) and one-time codes (TOTP); required per setting", ["FR-161"], ["UM-S1", "UM-S2"], "L", ["P29-01"], q=["Q15"])
t("P29-03", "Web sessions: expiry, idle timeout, secure cookies, CSRF, lockout after failed tries", ["FR-168", "FR-186"], ["UM-S2"], "M", ["P29-01"])
t("P29-04", "Loopback untrusted in multi-user mode; trust as a named service account, with the warning", ["FR-164", "NFR-21"], ["UM-S2"], "S", ["P29-03"])
t("P29-05", "Tailscale identity only for linked users", ["FR-165", "FR-17"], ["UM-S1"], "S", ["P29-03"])
t("P29-06", "gbi login (device code), expiring revocable tokens for CLI and MCP", ["FR-166"], [], "M", ["P29-03"])
t("P29-07", "Agent sessions get short-lived tokens; X-GBI-Persona ignored in multi-user mode; gbi users recover", ["FR-167"], [], "M", ["P29-06"])
t("P29-08", "Tests, docs, CHANGELOG, release", ["FR-133", "FR-130"], ["UM-S1"], "M", ["P29-02", "P29-04", "P29-05", "P29-07"])

# P30: People & access, Take 1.
t("P30-01", "People & access → Users: statuses and filters, invitations by email, bulk actions; gbi users", ["FR-169", "FR-170"], ["UM-1a"], "L", ["P29-08"], q=["Q15"])
t("P30-02", "A person's page: status, sign-in methods, projects and roles with where each came from, sessions and tokens, access history", ["FR-170"], ["UM-1d"], "M", ["P30-01"])
t("P30-03", "Groups and service accounts", ["FR-171"], ["UM-1a"], "M", ["P30-01"])
t("P30-04", "Custom access roles: tick permissions, start from a role, gate types, versions, who it affects; risky permissions marked", ["FR-174", "FR-175"], ["UM-1b"], "L", ["P30-03"])
t("P30-05", "Project → Access tab and Check access", ["FR-178", "FR-177"], ["UM-1c"], "M", ["P30-04"])
t("P30-06", "Request access from a link; requests in the access managers' inbox on desktop and phone", ["FR-181"], ["UM-2d"], "M", ["P30-05"])
t("P30-07", "Nobody signs off their own work; more than one approver per project", ["FR-180"], [], "S", ["P30-05"])
t("P30-08", "Tests, docs, CHANGELOG, release", ["FR-133", "FR-130"], ["UM-1a"], "M", ["P30-02", "P30-06", "P30-07"])

# P31: harness credentials.
t("P31-01", "Shared or Per user setting; Settings → Harnesses shows the credential for each harness and model", ["FR-200", "FR-202"], ["UM-D4"], "M", ["P28-08"])
t("P31-02", "Harness signed in inside a user's own config folder, guided in a herdr pane", ["FR-201"], ["UM-D3", "UM-D4"], "M", ["P31-01"])
t("P31-03", "API keys per harness and model, encrypted, never shown again or logged", ["FR-201", "NFR-22"], ["UM-D4"], "M", ["P31-01"])
t("P31-04", "Work runs on the starter's credentials; automatic dispatch only for projects naming whose credentials pay", ["FR-203"], [], "M", ["P31-02", "P31-03"])
t("P31-05", "Doctor tests each credential and that the configured model starts without a menu", ["FR-202", "FR-205", "FR-07b"], ["UM-D4"], "S", ["P31-04"])
t("P31-06", "Tests (harness check in Per user mode), docs, CHANGELOG, release", ["FR-133", "NFR-23"], [], "M", ["P31-05"])

# P32: sign-off routes, access review (from Take 3, on Take 1's roles and groups).
t("P32-01", "Sign-off routes per gate type: groups or roles, in order or together, how many from each", ["FR-182", "FR-115"], ["UM-3c"], "L", ["P30-08"])
t("P32-02", "Sign-offs tied to versions; changes withdraw them; delegation for a period", ["FR-183", "FR-184"], ["UM-3c"], "M", ["P32-01"])
t("P32-03", "Access review: findings, decisions with reasons, CSV export", ["FR-187"], ["UM-3d"], "M", ["P30-08"])
t("P32-04", "Temporary elevation with a reason, approved and ended by itself (later phase)", ["FR-189"], [], "M", ["P32-03"])
t("P32-05", "Project-defined roles in docs/gbi.toml under a ceiling: not planned (Take 2); revisit only if asked", ["FR-176"], ["UM-2c"], "S", [])
t("P32-06", "Tests, docs, CHANGELOG, release", ["FR-133", "FR-130"], [], "M", ["P32-02", "P32-03"])

# P33: OIDC.
t("P33-01", "OIDC settings: issuer, client, secret, connection test, redirect URL", ["FR-162"], ["UM-S2"], "M", ["P29-08"], q=["Q14"])
t("P33-02", "Sign in with SSO; match by verified email; groups from a claim; local accounts off with break-glass", ["FR-162", "FR-171"], ["UM-S1"], "L", ["P33-01"], q=["Q14"])
t("P33-03", "Tests, docs, CHANGELOG, release", ["FR-133", "FR-130"], [], "S", ["P33-02"])

old = {x["id"]: x for x in D["tasks"]}
for x in T:
    if x["id"] in old:
        x["status"] = old[x["id"]]["status"]
        x["done"] = old[x["id"]].get("done") or x["done"]
oldph = {p["id"]: p for p in D["phases"]}
for p in PHASES:
    for k in ("gateApproved", "gateApprovedBy", "gateEvidence"):
        if k in oldph.get(p["id"], {}):
            p[k] = oldph[p["id"]][k]
keep = lambda pid: int(pid[1:]) < 26
D["phases"] = [p for p in D["phases"] if keep(p["id"])] + PHASES
D["tasks"] = [x for x in D["tasks"] if keep(x["phase"])] + T
D["requirements"], D["questions"] = reqs, questions
D["updated"] = datetime.date.today().isoformat()

ids = {x["id"] for x in D["tasks"]}
errs = []
for x in D["tasks"]:
    errs += [f"{x['id']} bad dep {d}" for d in x["deps"] if d not in ids]
    errs += [f"{x['id']} unknown req {r}" for r in x["reqs"] if r not in reqs]
    errs += [f"{x['id']} unknown question {q}" for q in x["q"] if q not in questions]
missing = sorted(set(reqs) - {r for x in D["tasks"] for r in x["reqs"]})
if missing:
    errs.append("uncovered: " + ", ".join(missing))
if errs:
    print("\n".join(errs)); sys.exit(1)

open(DATA, "w").write(head + "window.TRACKER = " + json.dumps(D, indent=1, ensure_ascii=False) + ";\n")
done = sum(1 for x in D["tasks"] if x["status"] == "done")
print(f"ok: {len(D['tasks'])} tasks ({len(T)} rev 12–13), {len(reqs)} requirements covered, {len(questions)} questions, {done} done")
