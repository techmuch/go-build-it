# Access control

How a project decides who may do what: users, groups, roles and permissions, and where the check is made. The aim is a model that anyone can explain in one sentence: *a principal may do something only when a role they hold, in that scope, contains the permission.*

- Every rule has an ID. Review findings and project overrides cite it (`PERM-31`).
- **SHALL** rules block a merge. **SHOULD** rules may be skipped with a reason in the pull request.
- Words used here: a **principal** is anyone who acts (a person, an AI member or a service). A **permission** is one action. A **role** is a named set of permissions. A **group** is a named set of principals. A **scope** is where a grant applies (everything, or one tenant or project).

## The model

- **PERM-01** SHALL: Four things are kept apart: principals, groups, roles and permissions. A principal belongs to any number of groups. A role is granted to a principal or to a group. A role contains any number of permissions.
- **PERM-02** SHALL: A permission is never granted directly to a principal or a group. It is reached only through a role. This keeps one question, "which roles does this person hold?", enough to explain all their access.
- **PERM-03** SHALL: Roles say what a job needs (`reviewer`, `billing-admin`). Groups say who belongs together (`engineering`, `region-east`). A group is never used as a permission check, and a role is never used as a mailing list.
- **PERM-04** SHALL: Roles are flat. A role does not inherit from another role, and a group does not contain another group. A role that needs another's permissions lists them. Effective access is then one step from grant to permission, and there are no cycles to detect.
- **PERM-05** SHALL: There are no deny rules. Access is the union of what a principal's roles allow, and anything not allowed is denied. A project that adds deny rules, through an override, makes deny win over allow in every case.
- **PERM-06** SHALL: A new principal, a new group and a new role start with no permissions.

## Permissions

- **PERM-10** SHALL: Permissions are defined in code, in one table, and are never created, renamed or deleted at run time. Roles, groups and grants are data that administrators change.
- **PERM-11** SHALL: A permission is named `resource.action` in lower case (`project.read`, `review.approve`, `role.grant`). Each has a one-line description that an administrator can read. There are no wildcards (`project.*`, `*`) in code or in stored data.
- **PERM-12** SHALL: Permissions are fine-grained and independent: holding one never implies another. Reading and changing are separate permissions, and so are changing and deleting.
- **PERM-13** SHALL: Every operation in the route table declares exactly one permission, or is marked `public` or `authenticated` on purpose. A test fails for a route that declares nothing, and for a permission that no route or service uses.
- **PERM-14** SHALL: Renaming or removing a permission is a breaking change. It ships with a migration that updates every role, and a change log entry.
- **PERM-15** SHALL: At start-up the application compares the permissions stored in roles with the table in code. An unknown permission stops the start with a message that names it. It is never ignored.
- **PERM-16** SHALL: A permission can be marked **human only** (approving a review, a gate or a release; granting roles). Such a permission cannot be given to an AI member or a service, and the check refuses it even if the data says otherwise.

## Roles, groups and grants

- **PERM-20** SHALL: A grant records: the role, who receives it (a principal or a group), the scope, who granted it, when, why, and an optional expiry. The reason is required.
- **PERM-21** SHALL: Every grant has a scope: `global`, or one tenant or project. A role granted on one project gives nothing on another. Global grants are rare and are listed separately in every access report.
- **PERM-22** SHALL: Built-in roles are defined in code and seeded by a migration. They cannot be deleted, and their changes ship with a release. Administrators create custom roles from the fixed permissions.
- **PERM-23** SHALL: The one built-in `owner` role is defined in code as every permission that is not excluded for its kind of principal. Every other role lists its permissions explicitly, so a new permission is granted to nobody else until someone adds it.
- **PERM-24** SHALL: Nobody can grant a role that contains a permission they do not hold themselves in that scope, and nobody can change their own grants.
- **PERM-25** SHALL: The last owner cannot be removed, disabled or demoted. The operation fails with a message that says why.
- **PERM-26** SHALL: Duties that check each other are separate permissions held by different principals: the author of a change cannot approve it, and whoever manages roles cannot erase the record of doing so.
- **PERM-27** SHALL: A grant with an expiry stops working at that time without any job having to run: the check compares the time. Temporary and emergency access always has an expiry.
- **PERM-28** SHALL: Groups from an identity provider are mapped to roles in one configuration table. Group names from a token are never compared in code.
- **PERM-29** SHALL: AI members and services are principals with their own identity and their own grants. They never act with a person's grants, and they start from the smallest role that lets them do their task.

## The check

- **PERM-30** SHALL: One function makes every decision: `Authorize(ctx, permission, object)`. It reads the principal from the context and returns nil or a typed "forbidden" error. No other code inspects roles or groups to decide access.
- **PERM-31** SHALL: Code asks for a permission, never for a role. `if user.Role == "admin"` is a defect.
- **PERM-32** SHALL: The check is made on the server for every operation, by the code that mounts the route table, before the handler runs. The HTTP API, the CLI and the MCP server share that path, so none can skip it.
- **PERM-33** SHALL: A permission on a kind of thing is not permission on every instance. Where access depends on the object (its project, tenant or owner), the service calls `Authorize` with the object, after loading it and before using it. Lists are filtered by the same rule in the query, not afterwards in memory.
- **PERM-34** SHALL: Ownership rules ("authors may edit their own drafts") are written once, next to the permission table, as a named condition on a permission. They are not scattered through handlers.
- **PERM-35** SHALL: The check fails closed. If the principal, the grants or the object cannot be loaded, the answer is no.
- **PERM-36** SHALL: A request with no valid identity gets `401`. A request from a known principal without the permission gets `403`, and the message names the missing permission. Where the existence of an object is itself restricted, the answer is `404`.
- **PERM-37** SHALL: Background work acts as a named principal with its own grants. No code path runs "as system" with every permission to avoid a check.

## Sessions, tokens and caching

- **PERM-40** SHALL: A token or session carries identity only. Permissions are worked out on the server from current data. They are not stored in a JWT, a cookie or the browser, so a revoked grant cannot live on in a token.
- **PERM-41** SHALL: Effective permissions may be cached in memory per principal. Every change to a role, group, membership or grant advances one version counter, and a cache entry from an older version is discarded before use.
- **PERM-42** SHALL: A revoked grant, a removed membership and a disabled principal take effect on the next request. A test proves this for each.
- **PERM-43** SHOULD: The check adds no database query on a warm cache, and the application reports cache hits, misses and denials.

## What people see

- **PERM-50** SHALL: The UI learns the current principal's effective permissions from one endpoint and hides or disables controls the principal cannot use. This is for clarity only. It is never the enforcement (PERM-32).
- **PERM-51** SHALL: Anyone with the right permission can ask why: `access explain <principal> <permission> [--on <object>]` answers allowed or denied, and shows the path: grant, role, group and scope.
- **PERM-52** SHALL: An access report lists, for a scope, every principal with their roles, how each role reached them (directly or through which group), who granted it, when and why. It is available from the UI and the CLI, and as JSON.
- **PERM-53** SHALL: The role editor shows each permission's description and, before saving, the principals whose access will change.

## Records

- **PERM-60** SHALL: Every change to a role's permissions, a group's members or a grant is a history event with who made it, what changed (before and after), when and why.
- **PERM-61** SHALL: Every denial is logged with the principal, the permission, the object and the request ID. Repeated denials for one principal are counted so they can raise an alert.
- **PERM-62** SHALL: The state of access at any past time can be rebuilt from the history.

## Tests

- **PERM-70** SHALL: A table test runs every route against every built-in role and compares allowed and denied with a reviewed table in the repository. Adding a route or changing a role changes that table in the same pull request, where a reviewer sees it.
- **PERM-71** SHALL: Every route has a test that calls it with no identity and with a principal that has no roles, and expects `401` and `403`.
- **PERM-72** SHALL: Every operation on an object has a test in which a principal with the permission in one scope tries the same operation on an object in another scope, and is refused.
- **PERM-73** SHALL: Tests cover: expiry, revocation taking effect at once, the last-owner rule, the no-self-grant rule, human-only permissions refused for AI members, and an unknown stored permission stopping start-up.
- **PERM-74** SHALL: The same checks are run through the CLI and MCP paths at least once each, to prove they share the enforcement path.

## Why

- **Explainable.** Flat roles, no deny rules and no direct permissions mean a person can read a principal's roles and know what they can do. Models with inheritance and exceptions are where access surprises come from.
- **One place to be right.** A single function and a single mount point mean a missing check is a missing table entry that a test catches, not a forgotten line in one of many handlers.
- **Revocation that works.** Keeping permissions out of tokens and tying caches to a version counter means removing access removes it now.
- **Safe with agents.** AI members get their own identities, small roles and no human-only permissions, so a person stays the one who approves.
