# Software hardening

How a project is built so that an attacker who gets in can do little, and a mistake stays small. It assumes a breach will happen and limits what it reaches. Two more standards complete the set: *AI layer security* (`AIS-`) and *Operational resilience* (`OPS-`).

- Every rule has an ID. Review findings and project overrides cite it (`HARD-21`).
- **SHALL** rules block a merge. **SHOULD** rules may be skipped with a reason in the pull request.

## Threat model

- **HARD-01** SHALL: `docs/security/threat-model.md` exists before the first release. It lists what is worth protecting, who might attack it, every way in (each listener, input, file, dependency and AI input), and what stops each attack or limits its damage.
- **HARD-02** SHALL: A change that adds a way in (a listener, an input, an integration, a dependency, an AI tool) updates the threat model in the same pull request.
- **HARD-03** SHALL: Every protection the threat model relies on has a test that fails when the protection is removed.

## Smallest attack surface

- **HARD-10** SHALL: The software ships safe by default. A fresh install listens on loopback only, has no default account or password, and has every optional feature off.
- **HARD-11** SHALL: Each listener, port, endpoint and command is there because a requirement needs it. Debug endpoints, profilers, sample data and test hooks are not in release builds, and a test checks the route table of a release build for them.
- **HARD-12** SHALL: The release contains only what runs: one static binary and its embedded files. No shell, compiler, package manager or interpreter is shipped with it.
- **HARD-13** SHALL: Features that widen access (remote access, external integrations, AI tools that act) are turned on by an explicit setting, and turning one on is recorded in the history.
- **HARD-14** SHALL: Unused code, routes, flags and dependencies are removed, not left disabled.

## Zero trust between parts

- **HARD-20** SHALL: No request is trusted because of where it comes from. Every call between services, and every call from a person, carries an identity that the receiver verifies cryptographically (mutual TLS or a signed, short-lived token).
- **HARD-21** SHALL: Every service, job and AI member has its own identity and the fewest permissions that let it work (see the *Access control* standard). Identities are not shared.
- **HARD-22** SHALL: Each service may call only the services it is declared to need. The declaration is in the repository, the network policy is generated from it, and anything else is refused.
- **HARD-23** SHALL: Credentials are short-lived and issued at run time by the platform. Long-lived keys are the exception, are listed in the threat model, and can be rotated without a release.
- **HARD-24** SHALL: All traffic is encrypted in transit, inside the network as well as outside it.
- **HARD-25** SHALL: A compromise of one part is assumed in design: for each service, the threat model states what an attacker holding it could reach, and that list is kept short.

## Confinement at run time

- **HARD-30** SHALL: The process runs as an unprivileged user, with no added operating-system capabilities and no way to gain more.
- **HARD-31** SHALL: The file system is read-only in production, apart from the data directories the software declares. Nothing executable can be written and then run.
- **HARD-32** SHALL: The process has limits on memory, CPU, open files and child processes, and the software behaves sensibly when it reaches them.
- **HARD-33** SHALL: Outbound network access is limited to the destinations the software declares. A fetch of a URL supplied by a user or an AI model is checked against an allowlist and can never reach internal addresses or cloud metadata services.
- **HARD-34** SHALL: Anything that runs code or commands the project did not write (plug-ins, agent tools, user scripts, document previews) runs in a sandbox with its own, smaller permissions, and is treated as hostile.
- **HARD-35** SHOULD: Where data is highly sensitive and the platform offers it, keys and the most sensitive processing are placed in hardware-backed isolation (a secure enclave or confidential virtual machine). The threat model says what this protects against.

## Input, output and errors

- **HARD-40** SHALL: All input is validated at the boundary against an allowlist of type, length, range and format, on the server. Invalid input is rejected, not repaired.
- **HARD-41** SHALL: Every input has a size limit, every list has a page limit, and every operation has a time limit. Nothing a caller sends can make the software allocate or loop without bound.
- **HARD-42** SHALL: Data is never mixed with code: SQL uses placeholders, commands use argument lists, file paths are confined to their root, and output is encoded for where it goes.
- **HARD-43** SHALL: Parsers of complex or untrusted formats have fuzz tests (`go test -fuzz`), and the pipeline runs them for a fixed time on every change to the parser.
- **HARD-44** SHALL: The software fails closed. When a check cannot be made, the answer is no. An error tells the user what to do and tells an attacker nothing: no stack trace, query, path or version.
- **HARD-45** SHALL: Requests are rate-limited per identity and per source, and repeated failures slow the caller down.

## Secrets and data

- **HARD-50** SHALL: Secrets come from the platform's secret store at run time. They are never in the repository, an image, a build argument, a log, an error, a URL or a prompt.
- **HARD-51** SHALL: A secrets scanner runs on every change and over the history. A secret that reaches the repository is treated as compromised: it is rotated, not just deleted.
- **HARD-52** SHALL: Sensitive data is encrypted at rest, and the software collects and keeps only what a requirement needs. Each kind of data has a stated retention period after which it is deleted.
- **HARD-53** SHALL: Cryptography comes from the standard library or a validated module. The project implements no algorithm or protocol of its own.

## Supply chain

- **HARD-60** SHALL: Dependencies are few, pinned with checksums, and each is justified in an ADR. Builds fetch nothing that is not pinned.
- **HARD-61** SHALL: Every release has an SBOM generated by the pipeline from the built artifact, covering every direct and nested component, stored and signed with the release.
- **HARD-62** SHALL: The pipeline runs static analysis, a security linter, a reachable-vulnerability scan and a secrets scan on every change, and a finding blocks the merge.
- **HARD-63** SHALL: Released artifacts are scanned again every day against their SBOM, so a newly published vulnerability opens a task without anyone having to look.
- **HARD-64** SHALL: A known vulnerability that the code can reach is fixed within a stated time: 7 days for critical, 30 for high, 90 for the rest. A project may set shorter times.
- **HARD-65** SHALL: Releases are built only by the pipeline from a protected branch, are signed, and are verified before they are installed or deployed. The same commit builds the same binary.
- **HARD-66** SHALL: Every change is reviewed by a person who did not write it. A review by an AI member is useful evidence, and it never replaces that person.

## Why

- **Assume breach.** Perimeters fail. What decides the outcome is how little an attacker can reach from where they land, so every rule here shrinks that reach.
- **Safe defaults last.** Most systems run with the settings they were installed with. If the default is safe, most installations are.
- **Tested protections stay.** A protection without a test is removed by the next refactoring, whether a person or an AI member does it.
