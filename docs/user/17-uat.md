section: UAT walkthrough
reqs: FR-132, FR-133
# UAT walkthrough by role

The walkthrough is automated in `web/e2e/uat.spec.ts` (Playwright). It runs against a demo seeded with `scripts/demo.sh`. The demo starts an isolated herdr (never your own sessions), and its members run a scripted agent inside herdr, so no subscription is used. It also regenerates every screenshot in these docs.

**Owner (desktop):**
1. Catalog: projects that need you come first.
2. Project overview, then the roadmap item and its gate.
3. Artifacts and values; edit with a team member and accept a proposal.
4. Questions: open an attachment in the viewer.
5. Sessions: watch a member's live terminal, take control and hand back.
6. Queue: work held by a gate.
7. Option set: combine aspects from two options.
8. Dress rehearsal: findings from several reviewers.
9. Work review: the assessment.
10. Persona archetypes: change review and coach.
10b. Project team: tuning, the diff against the archetype, promote.
10c. A member waiting for you: answer it in its live terminal and the task carries on.
11. Standards: edit with Lens.
12. History, then an event detail.
13. Settings.
14. Docs; ⌘K palette.

**Owner (mobile):** M1 home → M3 inbox → M4 answer → M5 review → M6 rehearsal → M7 options → M8 sessions → M9 more → M10 terminal.

**Reviewer (a teammate over the tailnet):**
- The identity badge shows their login.
- Their findings are attributed to them.
- Gate approval is refused.

**Project Manager (an agent through the API):**
- Queues tasks and asks questions.
- Cannot answer questions.

Run it:

```
GBI_HOME=$(mktemp -d) PORT=7431 scripts/demo.sh
cd web && GBI_E2E_URL=http://127.0.0.1:7431 npx playwright test
```
