section: Using go-build-it
reqs: FR-05, FR-06, FR-63, FR-90
# Settings and updates

Settings cover:
- **harnesses:** detection, enable or disable each one, and whether its context usage is measured or estimated
- **access:** Tailscale
- **forges:** GitHub and GitLab
- **dispatch**
- **experimental features**
- **updates**

The git workflow is a standard, not a setting.

![Settings](screenshots/settings.png)

**Updates:** new releases come from GitHub Releases. Each is checked against `SHA256SUMS` and its Ed25519 signature, installed only while no sessions are running, and can be rolled back (ADR-004).
