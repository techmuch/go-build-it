# React frontend

How a project's web UI is built: a React and TypeScript single-page app, embedded in the Go binary, that talks only to the project's own HTTP API.

- Every rule has an ID. Review findings and project overrides cite it (`FE-41`).
- **SHALL** rules block a merge. **SHOULD** rules may be skipped with a reason in the pull request.
- A project that needs something different overrides the rule, with a reason. It does not quietly ignore it.

## Stack and dependencies

- **FE-01** SHALL: React with function components and hooks, TypeScript, built with Vite. The build output is embedded in the Go binary (`embed.FS`), so there is one thing to install.
- **FE-02** SHALL: TypeScript runs with `strict`, `noUnusedLocals` and `noUnusedParameters`. `tsc --noEmit` passes.
- **FE-03** SHALL: No CSS framework, component library, state-management library, data-fetching library or router package. The app's own small modules do these jobs (FE-10).
- **FE-04** SHALL: A new runtime dependency is recorded in an ADR: what it does, why the platform or the app's own code is not enough, its licence and what it adds to the bundle.
- **FE-05** SHALL: `npm ci` installs from a committed lock file, and `npm audit --omit=dev` reports nothing high or critical.
- **FE-06** SHALL: ESLint (with the React hooks rules) and Prettier run in `make test` and in CI. Lines stay under 120 characters and each statement has its own line.

## Layout

- **FE-10** SHALL: One module per job: `router` (paths), `http` (the fetch wrapper), `api.gen` (the generated client), `live` (server events and data loading), `keys` (keyboard), `ui` (shared components), `shell` (navigation and page frame). Pages live in `pages/`, one file per screen or closely related group.
- **FE-11** SHALL: All routes are declared in one table, in one file. Every view has its own URL, including the selected item and the open tab, so a link, a reload and the Back button all return to the same place.
- **FE-12** SHALL: Desktop and mobile are separate route trees (`/` and `/m/`) that share the client, the hooks and the tokens. A phone screen is designed for a phone. It is not the desktop page squeezed.
- **FE-13** SHALL: A component is added to `ui` when a second page needs it. Before then it stays in its page.
- **FE-14** SHOULD: A file stays under 400 lines and a component under 150. Split by concept.
- **FE-15** SHOULD: Each page is loaded on demand (`lazy`), and the first-load JavaScript stays under 200 kB gzipped. CI reports the size.

## Data and state

- **FE-20** SHALL: Data comes only from the HTTP API, through the generated client. No component calls `fetch` itself, and the client file is never edited by hand.
- **FE-21** SHALL: The server is the only source of truth. The UI does not keep its own copy of server data beyond the component that shows it. There is no global store.
- **FE-22** SHALL: A view loads its data with one hook that returns `data`, `error` and `loading`, and that reloads when a matching live event arrives. Reloads are debounced.
- **FE-23** SHALL: Live updates come over one shared Server-Sent Events connection. The UI shows whether it is connected, and nothing polls on a timer.
- **FE-24** SHALL: Every write goes through one action hook. It disables the control while the call runs, reports success or the server's error message in a toast, and never fails silently.
- **FE-25** SHALL: Every view handles four states on purpose: loading, empty (with what to do next), error (with the message) and ready.
- **FE-26** SHALL: An error boundary wraps each route, so one broken view shows a message and the rest of the app keeps working.
- **FE-27** SHALL: `localStorage` holds only per-device preferences (theme, a hidden panel). Every access is in `try`/`catch`, and the app works without it.
- **FE-28** SHALL: No `any` outside the generated client. API responses have declared types, and the generated client returns them once the route table declares response types.
- **FE-29** SHALL: An effect cleans up what it starts: listeners, timers, sockets and requests whose answer is no longer wanted.

## Styling

- **FE-30** SHALL: One token stylesheet. Colours, shadows and radii are CSS custom properties on `:root`, and every other rule uses the tokens. No hex colour appears outside the token block.
- **FE-31** SHALL: Light and dark themes are both complete. The theme follows the system setting and can be forced with `data-theme`.
- **FE-32** SHALL: Components are styled with classes from the stylesheet. An inline `style` is only for a value computed at run time (a width from data, a scale factor).
- **FE-33** SHALL: A status always maps to the same colour, through one table. Colour is never the only signal: the status word or an icon is shown too.
- **FE-34** SHALL: No view scrolls sideways at any supported width. The end-to-end tests assert this on every screenshot.
- **FE-35** SHALL: Animations respect `prefers-reduced-motion`.

## Keyboard and accessibility

- **FE-40** SHALL: Everything that can be done with a pointer can be done from the keyboard. The conventions are implemented once, in `keys`, and pages opt in with attributes. They do not write their own key handlers for the same job.
- **FE-41** SHALL: A real `<button>` or `<a href>` is used wherever one fits. A row or card that acts like one gets the shared helper, which adds the role, `tabIndex`, Enter and Space.
- **FE-42** SHALL: Focus is always visible (`:focus-visible`). The first Tab on a page offers *Skip to content*.
- **FE-43** SHALL: A dialog takes focus when it opens, keeps Tab inside, closes with Esc and gives focus back. It has `role="dialog"`, `aria-modal` and a label. The browser's `alert`, `confirm` and `prompt` are not used.
- **FE-44** SHALL: Single-key shortcuts never fire while the user is typing in a field.
- **FE-45** SHALL: Every shortcut is listed in an in-app help dialog and in the user docs, and a test fails when the two lists differ.
- **FE-46** SHALL: Every input has a label, every icon-only button has an `aria-label`, every image has `alt`, and every frame has a `title`. Text contrast meets WCAG AA in both themes.
- **FE-47** SHALL: An automated accessibility check (axe) runs in the end-to-end tests on every view and reports no serious or critical violation.

## Working with people and agents

- **FE-50** SHALL: The UI always shows who is acting (the signed-in identity and how they reached the app) and who did each thing: a person, an AI member or the system.
- **FE-51** SHALL: Anything waiting on a person is collected in one inbox with a count in the navigation, and items that block work are marked.
- **FE-52** SHALL: Every action in the UI has a CLI command, and the command palette shows it. A person can learn the command an agent would run by looking at the screen.
- **FE-53** SHALL: An action that costs money, cannot be undone or is visible outside the machine asks first, in an in-app dialog that says what will happen.
- **FE-54** SHALL: Each view carries the ID of the mockup view it implements (`v03`), so a built screen can be compared with its mockup and traced to its requirements.

## Security

- **FE-60** SHALL: Markdown and HTML that came from a user, a repository or an AI member are sanitised before they are put in the page. `dangerouslySetInnerHTML` appears in one component only, and that component sanitises.
- **FE-61** SHALL: Project documents and mockups are shown in a sandboxed `<iframe>` without `allow-same-origin`, served from a route that also sends a `sandbox` Content-Security-Policy. They cannot reach the app's API as the user.
- **FE-62** SHALL: Values placed in a URL are encoded (`encodeURIComponent`). The generated client does this for path parameters.
- **FE-63** SHALL: No secret is kept in the bundle, in `localStorage` or in a URL that the app generates.
- **FE-64** SHALL: The app loads no script, font or style from another origin. It works with no internet connection.

## Tests

- **FE-70** SHALL: Every view has a Playwright end-to-end test, run at a desktop and a phone viewport, against a seeded demo instance whose AI members are scripted. The tests need no credentials and cost nothing.
- **FE-71** SHALL: The tests are organised as a walkthrough by role (owner, reviewer, an agent through the API), and each step names the requirement it shows.
- **FE-72** SHALL: The screenshots in the user docs are written by the end-to-end tests, never by hand. Screens that are compared with mockups are saved under their view ID.
- **FE-73** SHALL: Tests find elements the way a user does: by role, label or visible text. `data-testid` is for elements with no stable name. CSS class selectors are a last resort.
- **FE-74** SHALL: A test waits for a condition (`expect(…).toBeVisible()`), never for a fixed time.
- **FE-75** SHALL: The keyboard conventions have their own end-to-end spec: focus order, arrow keys, dialogs and focus return.
- **FE-76** SHALL: Pure logic (the route matcher, formatters, parsers) has unit tests that run without a browser.
- **FE-77** SHALL: CI runs the type check, the linter, the unit tests and the end-to-end tests on every push. A pull request that changes a view includes its new screenshots.
- **FE-78** SHALL: A bug fix starts with a test that fails without the fix.

## Before asking for review

All of these pass, and `make test` and CI run them:

```
npm ci
npx tsc --noEmit -p .
npx eslint . && npx prettier --check .
npm run test                 # unit tests
npm run build                # reports bundle size
npx playwright test          # against the seeded demo; rewrites the docs screenshots
git diff --exit-code src/api.gen.ts   # after regenerating the client
```

## Why

- **One small stack.** With few dependencies and one module per job, an AI member can read the whole frontend, and patterns carry from one project to the next.
- **The server decides.** Loading through one hook and reloading on live events means the screen shows what people and agents have actually done, without each page inventing its own caching.
- **Evidence a person can judge.** Screenshots produced by the tests are the documentation, the review evidence and the comparison with the mockup, so a reviewer looks at the real screen instead of trusting a description.
- **Usable by everyone.** Keyboard and accessibility rules live in shared code and are tested, so they hold as agents add views.
- **Safe to show agent output.** Agents write Markdown and HTML. Sanitising and sandboxing keep that content from acting as the user.
