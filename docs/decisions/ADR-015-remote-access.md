# ADR-015: Remote access proves it works and never takes another service's address

Status: Accepted · decided by the owner 2026-11-12 · Tracker: P23-01 · Requirements: FR-20, FR-20a–FR-20d · Builds on [ADR-005](ADR-005-external-access.md)

## Context

go-build-it is reached from other devices through Tailscale Serve, and it trusts Tailscale's identity headers (ADR-005). The owner asked for a check that Serve is really working, that it is not clashing with other services on the same machine, and for a QR code to open it on a phone.

Before this decision:
- **"Serving" was a text search.** go-build-it reported itself served whenever its port number appeared anywhere in `tailscale serve status`. It never checked that the address answered, or that the answer came from this go-build-it.
- **Setup could overwrite another service.** "Set up Tailscale Serve" ran `tailscale serve --bg http://127.0.0.1:7420`, which takes the root of port 443. If another tool already served there, go-build-it silently replaced it. Any other tool doing the same knocks go-build-it off.
- **Nothing checked Funnel.** If Funnel were on for go-build-it's address, it would be on the public internet, protected only by headers that only Tailscale should set.

The owner's own machine shows the risk is real: four services are served from it (go-build-it on 443, plus three other programs on 3100, 8080 and 8443).

## Decision

1. **Read the whole Serve configuration** (`tailscale serve status --json`) and match go-build-it's rule exactly: a root handler that proxies to go-build-it's own loopback port.
2. **Prove the round trip.** Fetch `<tailnet URL>/api/system` and require the answer's data folder to be this go-build-it's. Another program, or another copy of go-build-it, fails the check.
3. **Private or fail.** Funnel on go-build-it's address is a failure, with the command that turns it off.
4. **Never take an address.** Setup uses the root of 443 only when it is free or already go-build-it's. Otherwise it names the service that owns it (the program listening behind it, when `lsof` can tell), proposes go-build-it's own HTTPS port (its local port when free, else the first free one from 8443), shows the exact command, and runs it only after the person confirms. Serving under a path such as `/gbi` was rejected: the web app assumes it is at the root.
5. **Stop removes only go-build-it's rules** (`tailscale serve --https=<port> off`).
6. **Owner only, recorded.** Only the owner may set up or stop serving; each change is a `tailscale.served` or `tailscale.stopped` history event.
7. **QR code made inside go-build-it** (`rsc.io/qr`, BSD, no dependencies of its own). It is shown only when the checks pass, in Settings and as half blocks in a terminal (`gbi tailscale qr`).
8. **Checked on demand, not in the background.** The checklist runs when Settings opens, on **Check again**, in `gbi tailscale status`, and as the doctor's `tailscale.serve` check. A background watcher was left for later.

## Consequences

- A broken or hijacked remote address is visible in Settings and in `gbi doctor`, instead of being found by someone failing to connect.
- Another tool on the machine can still take 443 from go-build-it, but the checklist shows it, and setup offers a port of go-build-it's own.
- The demo and UAT run against a scripted `tailscale`, so screenshots show a made-up tailnet and never the owner's.
