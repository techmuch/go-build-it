# ADR-005: External interface access in the MVP (Q1)

Status: Accepted · approved by the owner 2026-11-08 · Tracker: P0-07

The MVP refuses requests classified as *External* with HTTP 403 and listens on loopback only unless configured. Remote access goes through Tailscale Serve (FR-17, FR-20). If external access is wanted later, P9-07 adds a token challenge. This satisfies FR-18 and NFR-21: nothing is reachable externally without a challenge, because nothing external is allowed.
