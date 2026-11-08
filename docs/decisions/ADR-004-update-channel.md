# ADR-004: Auto-update channel and signing (Q6)

Status: Accepted · approved by the owner 2026-11-08 · Tracker: P0-06

Channel: GitHub Releases on `techmuch/go-build-it`, one asset per OS/arch plus `SHA256SUMS` and `SHA256SUMS.sig`. Signature: Ed25519 over `SHA256SUMS`; the public key is compiled into the binary. The updater downloads, verifies checksum and signature, waits until no sessions run, swaps the binary, keeps the previous one for rollback (FR-06). Homebrew tap can be added later as a second channel.
