# Operational resilience

How a project runs in production so that failures stay small, attackers cannot settle in, and the team finds out quickly. Part of the hardening set, with *Software hardening* (`HARD-`) and *AI layer security* (`AIS-`).

- Every rule has an ID. Review findings and project overrides cite it (`OPS-31`).
- **SHALL** rules block a merge. **SHOULD** rules may be skipped with a reason in the pull request.

## Declared, immutable infrastructure

- **OPS-01** SHALL: Everything that makes up production is declared in the repository: infrastructure, network policy, configuration, permissions and dashboards. Production is created from those files by the pipeline.
- **OPS-02** SHALL: Nothing is changed by hand in production. There is no interactive shell into a running instance, and a change is made by changing the files and redeploying.
- **OPS-03** SHALL: A running instance is never patched or reconfigured. It is replaced by a new one built from the new version.
- **OPS-04** SHALL: The pipeline compares production with what is declared at least daily. A difference raises an alert and is treated as a possible intrusion until explained.
- **OPS-05** SHALL: Emergency access exists, is time-limited, needs a second person's approval, is fully recorded, and ends with the instance being replaced.
- **OPS-06** SHALL: Environments are built from the same declarations and differ only in the values of declared settings. The artifact tested is the artifact deployed.

## Short-lived instances

- **OPS-10** SHALL: An instance holds no state that matters. State lives in declared data stores, so any instance can be destroyed at any moment without losing data.
- **OPS-11** SHALL: Instances have a maximum lifetime, 24 hours by default, after which they are replaced on a rolling basis without interrupting service. The project records its value and the reason.
- **OPS-12** SHALL: The software starts quickly, reports when it is ready, and shuts down cleanly on a signal: it stops taking new work, finishes or hands back what is in progress, and exits within a stated time.
- **OPS-13** SHALL: Work that outlasts an instance is saved as it goes and resumed by another instance.
- **OPS-14** SHALL: Software that must run as one long-lived process (a desktop or single-machine tool) meets the intent differently: it can be restarted at any time without losing work, and a restart restores a known-good state.

## Failing well

- **OPS-20** SHALL: Every call to something else (a service, a database, a model, a command) has a timeout, and the caller's deadline is passed along.
- **OPS-21** SHALL: Retries are limited in number, spaced with increasing and randomised delays, and used only for operations that are safe to repeat. Operations that change state carry an idempotency key.
- **OPS-22** SHALL: A dependency that keeps failing is left alone for a while (a circuit breaker) so it can recover and callers fail fast.
- **OPS-23** SHALL: For each dependency the design says what the product does without it: which features continue, which are reduced and what the user is told. Security checks are never the part that is skipped.
- **OPS-24** SHALL: Under more load than it can serve, the software refuses the excess early and clearly. It does not slow down for everyone until it falls over.
- **OPS-25** SHALL: One tenant, user or job cannot use up what the others need: queues, pools and quotas are per tenant or per kind of work.
- **OPS-26** SHALL: A deployment goes to a small share of instances first and is rolled back automatically when error or latency measures worsen. Rolling back is one command and is practised.
- **OPS-27** SHALL: Schema changes are compatible with both the previous and the next version of the software, so a rollback never needs a data repair.

## Proving it: failure injection

- **OPS-30** SHALL: Failure is tested on purpose. The test suite includes cases that kill the process mid-task, drop or slow a dependency, fill a disk and exhaust a pool, and each checks that the software recovers by itself with no data lost.
- **OPS-31** SHALL: Failure experiments run on a schedule in a pre-production environment that matches production. Each states what should happen before it runs, and a different result opens a task.
- **OPS-32** SHOULD: Once an experiment passes repeatedly in pre-production, it runs in production too, during staffed hours, with a limited blast radius and an automatic stop.
- **OPS-33** SHALL: Backups are taken on a schedule, are encrypted, are stored where a compromise of production cannot delete them, and are restored in a test at least every quarter. The measured restore time and data loss are recorded against the targets.
- **OPS-34** SHALL: The project states its recovery targets: how long an outage may last and how much recent data may be lost. The design and the tests are held to them.

## Seeing what happens

- **OPS-40** SHALL: The software emits structured logs, metrics and traces with a request ID that ties them together, and reports its health and readiness.
- **OPS-41** SHALL: Each service has a few stated service-level objectives (availability, latency, correctness) with alerts on them. An alert names what is wrong and links to what to do.
- **OPS-42** SHALL: Normal behaviour is recorded as a baseline, and departures from it raise an alert even when nothing has failed: unusual query volume, data leaving, CPU or memory, authentication failures, access denials, new outbound destinations, or AI spend and tool calls.
- **OPS-43** SHALL: Security events (logons, denials, permission changes, configuration changes) go to a store the application cannot alter.
- **OPS-44** SHALL: Detection is tested. A scheduled test performs a harmless version of each behaviour that should alert and fails if no alert arrives.
- **OPS-45** SHALL: Logs and telemetry never contain secrets or restricted data.

## Responding

- **OPS-50** SHALL: Every alert has a runbook in the repository. A runbook is steps a person or an AI member can follow, with the commands to run and what each result means.
- **OPS-51** SHALL: An AI member may investigate and propose during an incident. Actions that change production are taken by a person, or by automation that a person approved in advance for that exact case.
- **OPS-52** SHALL: A suspected compromise is handled by isolating and replacing the instances and rotating every credential they could reach, after capturing the evidence. Cleaning a compromised instance in place is not an option.
- **OPS-53** SHALL: Every incident ends with a written review without blame: what happened, how it was found, what limited or widened the damage, and tasks with owners. Each incident adds a test, an alert or a failure experiment.
- **OPS-54** SHALL: Incident response is rehearsed at least twice a year, including restoring from backup and rotating all credentials.

## Why

- **Nothing to settle into.** An attacker needs time and a place to stay. Instances that are replaced daily and cannot be changed by hand take both away.
- **Untested recovery does not work.** Backups, rollbacks and self-healing fail the first time they are needed unless they have been run on purpose.
- **Small failures.** Timeouts, limits and staged deployments turn an outage for everyone into an error for a few.
- **Behaviour, not signatures.** New attacks have no signature. Watching for departures from normal catches what a list of known threats cannot.
