# ADR-017: Per-user workspaces on a Linux server: OCI images with go-containerregistry, run by gVisor, no daemon

Status: **Proposed** · 2026-11-20 · owner's direction: gVisor + go-containerregistry · Tracker: P34 · Requirements: FR-207–FR-218, NFR-20, NFR-25 · Open: Q18–Q20 · Builds on [ADR-016](ADR-016-users-dev-mode-assistant.md)

## Context

go-build-it is becoming one shared server for a team (ADR-016). On that server, each person's Assistant and the AI members they direct run whatever a project brings: dependency installs, build scripts, test suites, code from external repositories. Today every agent runs on the host as the server's user (NFR-20 allowed this for one person on their own machine). On a shared server that means one person's agent can read another person's worktrees, harness credentials and tokens, and the server's own data.

The owner asked for each user to have their own container workspace on Linux, and does not want to depend on the Docker Engine.

The options looked at:

| Option | Daemon | Isolation | How go-build-it uses it | Cost |
|---|---|---|---|---|
| **containerd** Go client | containerd (lighter than Docker, still a system service) | container (shared kernel) | official Go client over gRPC; pulls images, snapshots, runs tasks | another service to install and keep running; rootless containerd is extra setup |
| **runc / libcontainer** embedded | none: go-build-it is the supervisor | container (shared kernel) | build an OCI bundle, start it with runc or libcontainer | go-build-it must unpack images and prepare root filesystems; needs root or rootless user namespaces |
| **gVisor** (`runsc`) | none | strong: a user-space kernel written in Go sits between the workload and the host kernel | an OCI runtime: given a bundle, `runsc run` starts it | slower on system-call-heavy work (builds, git, package installs), to be measured; some system calls unsupported |
| **Firecracker** microVM (Go SDK) | none (one VMM process per VM) | strongest: hardware virtualisation | the Go SDK drives the Firecracker process (Firecracker itself is written in Rust) | needs KVM (bare metal, or cloud VMs with nested virtualisation); needs a kernel and root-filesystem images, and tap networking |
| Rootless Podman | none | container | shell out to the `podman` CLI | an external tool and its versions to track |

For images, **go-containerregistry** (a pure-Go library) pulls standard OCI and Docker images from any registry and flattens their layers into a root filesystem, with no daemon. It works with any of the runtimes above.

## Decision (proposed)

1. **One runtime interface** in go-build-it: create, start, run, attach a terminal, stop, remove, report use (FR-208). Everything else in go-build-it (herdr, sessions, Courier) talks to workspaces through it.
2. **Images with go-containerregistry.** go-build-it pulls OCI images itself, verifies them by digest, and unpacks them into a per-user root filesystem with the user's persistent home folder mounted in. No Docker Engine and no containerd (FR-209).
3. **gVisor (`runsc`) is the default runtime.** go-build-it writes the OCI bundle and runs `runsc` directly, so there is no daemon. It gives strong isolation without KVM, which matters because agents run code nobody has read. gVisor is written in Go, like go-build-it.
4. **runc as the faster level** for projects an admin marks trusted, through the same interface and the same bundle.
5. **Firecracker later**, as a third level where KVM is available, for installations that need hardware isolation.
6. **herdr runs inside each workspace.** Each user's herdr session lives in their workspace; its socket is exposed only to go-build-it on the host, so watching, taking control and blocked detection keep working (FR-214).
7. **Network allow-list per workspace**: the forges, registries and harness providers the admin allows; nothing else on the host except go-build-it's API through the user's token (FR-211).
8. **Solo mode, macOS and Windows are unchanged** (FR-216).

## To prove before accepting (spike, P34-01)

- `runsc` runs a go-build-it workspace image rootless (or with the least privilege the host allows) on a stock Ubuntu and Debian server; what needs root, subordinate IDs or cgroup delegation.
- herdr, Claude Code and the other harnesses start and work inside gVisor; their terminals and transcripts are readable from the host.
- The cost of gVisor on a real task: `npm ci`, a test suite, `git` operations, compared with runc and with the host (NFR-25: within 10%, or a reasoned exception).
- Networking: the allow-list, DNS, and reaching go-build-it's API from inside.
- Start time of a stopped workspace (NFR-25: under 3 seconds).

If gVisor fails a check that matters, the fallback is runc as the default with gVisor per project, then containerd if go-build-it can't manage images and root filesystems reliably on its own.

## Consequences

- One person's agents can't read another person's code, credentials or tokens, or the server's data. Agents can run untrusted code with less risk to the server.
- No Docker Engine or containerd to install; the doctor checks what the host needs (FR-216).
- go-build-it takes on image handling and root-filesystem preparation, which is security-sensitive code: it uses the maintained libraries (go-containerregistry, runsc) rather than writing its own, and gets its own tests and review.
- Each workspace uses memory and disk even when idle; limits (FR-210) and stopping idle workspaces (FR-215) keep this bounded.
- A user's harness credentials live in their workspace's home folder, which fits Per user credentials (FR-200); in Shared mode the server's credentials must be made available to each workspace, read-only.
