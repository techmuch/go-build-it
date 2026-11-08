section: Getting started
reqs: FR-15, FR-16, FR-17, FR-18, FR-19, FR-20
# Access, identity and Tailscale

go-build-it listens on `127.0.0.1:7420` only. Who you are depends on how you reach it:

| How you connect | Challenge | Identity in history |
|---|---|---|
| This machine (loopback) | none | the default user (`default_user` in config) |
| Tailscale Serve | none: Tailscale already authenticated you | your Tailscale login, e.g. `maria@github` |
| Any other network interface | refused (ADR-005) | — |

**Set up Tailscale Serve:** Settings → Access → *Set up Tailscale Serve*, or `gbi tailscale setup`. go-build-it then answers at `https://<machine>.<tailnet>.ts.net`, and the badge in the top bar shows *Tailnet · you@login*.

**Owners and reviewers (ADR-006):** anyone who reaches the app can answer questions, add rehearsal findings and review work. Only owners approve gates and start releases. Add owners with `owners = ["david", "maria@github"]` in `config.toml`.

**External access** stays off. If you turn on `allow_external`, you must also set `external_token`; external requests then need `Authorization: Bearer <token>`.
