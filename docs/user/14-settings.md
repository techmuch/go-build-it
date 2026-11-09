section: Using go-build-it
reqs: FR-05, FR-06, FR-07, FR-63, FR-90
# Settings and updates

Settings cover:
- **herdr:** the supervised `gbi` session, its version and socket; **herdr setup** runs the doctor (FR-07)
- **harnesses:** detection, enable or disable each one, its herdr agent kind, and whether its context usage is measured (from the transcript) or unavailable
- **access:** Tailscale
- **forges:** GitHub and GitLab
- **dispatch**
- **experimental features**
- **updates**

The git workflow is a standard, not a setting.

![Settings](screenshots/settings.png)

**Updates:** new releases come from GitHub Releases. Each is checked against `SHA256SUMS` and its Ed25519 signature, installed whenever one is ready (herdr keeps members running while go-build-it restarts), and can be rolled back (ADR-004).
