# DoD audit logging

How the application records who did what, so that it meets the NIST SP 800-53 Audit and Accountability (AU) controls and the audit rules of the DISA Application Security and Development STIG. Part of the DoD set: see *DoD ATO readiness* for how rules, overrides and evidence work.

- Every rule has an ID (`AUD-12`). **SHALL** rules block a merge.
- STIG rule numbers are from ASD STIG V6R4. The release in the system profile governs.

## Two records, two purposes

- **AUD-01** SHALL: The **audit log** is a security record for the ISSO and the security operations centre. It is separate from the application log (for operators) and from the product history (for users). An event may appear in more than one, and none replaces another.
- **AUD-02** SHALL: Audit events are written through one package with one function. No other code formats an audit record.

## What is recorded

- **AUD-10** SHALL: These events are always audited, on success **and** on failure (AU-2, AU-12):
  - logon, logoff and every failed logon
  - session creation, renewal, timeout and destruction (V-222441 to V-222445)
  - acknowledgement of the notice-and-consent banner
  - every access denial
  - creating, changing, enabling, disabling and removing accounts and roles (V-222413 to V-222416)
  - granting, changing and removing privileges, and every use of a privileged function (V-222431, V-222450 to V-222459)
  - reading, exporting, changing and deleting restricted data
  - changing configuration, security settings or audit settings
  - starting and stopping the application and the audit function
  - every action taken by an AI member or automation, under its own identity
- **AUD-11** SHALL: The list of event types is one table in code. The user documentation lists the same types, and a test fails when the two differ.

## What each record contains

- **AUD-20** SHALL: Each record is one line of JSON, written with `log/slog`'s JSON handler. Field names are fixed and documented (AU-3).
- **AUD-21** SHALL: Every record has all of these fields:

| Field | Content |
|---|---|
| `time` | UTC, RFC 3339, to the millisecond (V-222446) |
| `event` | the event type from the table, as `noun.verb` |
| `outcome` | `success` or `failure`, and the reason for a failure |
| `user` | the unique identifier: EDIPI for a person, the workload identity for a service or AI member (V-222449) |
| `actor_kind` | `human`, `service` or `ai` |
| `source_ip` | the client address, and the forwarded address when behind an approved proxy (V-222448) |
| `target` | the resource acted on: its type and ID |
| `component` | the application, its version and the host or instance |
| `request_id` | an ID that ties the record to the application log |
| `session` | a one-way hash of the session identifier, never the identifier |

- **AUD-22** SHALL: Requests also record the HTTP method, the path without its query string, the user agent and the referrer (V-222447).
- **AUD-23** SHALL: A record with a missing mandatory field is a defect. A test builds every event type and checks every field.

## What is never recorded

- **AUD-30** SHALL: No log of any kind contains a password, a token, a session identifier, a key, a certificate's private part, PII beyond the user identifier, PHI, CUI content or classified content (V-222444).
- **AUD-31** SHALL: This is enforced by construction, not by care. Sensitive types implement `slog.LogValuer` and render as `[REDACTED]`, request and response bodies are never logged, and query strings are dropped.
- **AUD-32** SHALL: A test logs every sensitive type and searches the output for its value. Static analysis fails the build for a raw request body or header map passed to a logger.

## Reliability and protection

- **AUD-40** SHALL: When an audit record cannot be written, the action it describes is refused and an alert is raised. An AO may accept continued operation for availability, and that decision is recorded in the system profile (AU-5).
- **AUD-41** SHALL: Audit records are written to standard output or to the platform's collector, and forwarded off the host to the central log system named in the system profile. The application does not manage, rotate or delete audit files (AU-4, AU-9).
- **AUD-42** SHALL: The application has no function that changes or deletes an audit record, and the account it runs as cannot do so in the central system.
- **AUD-43** SHALL: Audit records are kept for the period the AO sets. The default is one year, and it is recorded in the system profile (AU-11).
- **AUD-44** SHALL: The host clock is synchronised to the authoritative time source the platform provides. The application does not use its own clock source (AU-8).
- **AUD-45** SHALL: Administrators cannot turn audit off for themselves. A change to audit settings is itself audited, before it takes effect.

## Review

- **AUD-50** SHALL: Records can be searched by user, event type, outcome, target and time range with the central system's tools, using the fixed field names (AU-6, AU-7).
- **AUD-51** SHALL: The application emits a count of failed logons, access denials and audit write failures, so the platform can alert on them.
- **AUD-52** SHOULD: A person with the audit-review role can see, inside the application, the recent audit events for a user or a record, read-only.

## Why

- **Accountability needs every field.** A record without the user, the outcome or the target cannot answer the question an investigator asks, and an assessor checks each field.
- **Logs leak.** Audit records are copied to many systems and read by many people. Keeping restricted data out by construction is the only approach that survives new code.
- **Agents act too.** When AI members and automation act under their own identities, the record shows which actions a person took and which a machine took.
