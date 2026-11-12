# DoD application security

What the application itself must do to meet the DISA Application Security and Development (ASD) STIG and the NIST SP 800-53 controls behind it. Part of the DoD set: see *DoD ATO readiness* for how rules, overrides and evidence work.

- Every rule has an ID (`SEC-21`). **SHALL** rules block a merge.
- STIG rule numbers (`V-222389`) are from ASD STIG V6R4. The release named in the system profile governs. Where the AO sets a different value, the AO's value is used and recorded.
- Every rule here has an automated test, and the test names the rule ID.

## Identity and authentication

- **SEC-01** SHALL: Every request is authenticated, including requests from the same machine. There is no anonymous or "trusted local" access. The only exceptions are a health check that returns no data and the notice-and-consent banner (IA-2).
- **SEC-02** SHALL: People sign in with a DoD-approved credential: a CAC or PIV certificate, either checked directly (mutual TLS) or through the identity provider named in the system profile (OIDC or SAML). The application stores no passwords.
- **SEC-03** SHALL: With mutual TLS, the certificate chain is validated to the DoD PKI roots, revocation is checked (OCSP or CRL) and a certificate that cannot be checked is refused. The user is identified by EDIPI, never by the name on the certificate.
- **SEC-04** SHALL: With an identity provider, the token's signature, issuer, audience and validity period are verified on every request, with a FIPS-approved asymmetric algorithm (RS256, PS256, ES256 or EdDSA). `none` and shared-secret algorithms are refused.
- **SEC-05** SHALL: Services and automation authenticate with their own certificate or workload identity. No account is shared between people, between services or between AI members.
- **SEC-06** SHALL: Three consecutive failed logons within 15 minutes lock the account until an administrator unlocks it through the approved process (AC-7, V-222432). When the identity provider enforces this, the control is recorded as inherited.
- **SEC-07** SHALL: An account unused for 35 days is disabled, and a temporary account is removed after 72 hours (AC-2, V-222411, V-222409).

## Notice and consent

- **SEC-10** SHALL: The Standard Mandatory DoD Notice and Consent Banner is shown before access is granted, with its exact approved text. It stays on screen until the user acknowledges it (AC-8, V-222434, V-222435).
- **SEC-11** SHALL: After logon the user sees the date and time of their last successful logon (AC-9, V-222437).
- **SEC-12** SHALL: Every screen and every export shows the classification or CUI marking of what it contains, top and bottom. Stored and transmitted data carries the same marking (AC-16, V-222393 to V-222395).

## Sessions

- **SEC-20** SHALL: Session state is held on the server. The browser holds only an opaque, random identifier of at least 128 bits from the validated random source, in a cookie set `Secure`, `HttpOnly` and `SameSite=Strict`. It never appears in a URL or a log.
- **SEC-21** SHALL: A session ends after 15 minutes without activity, or 10 minutes for a user with administrative rights. The user is logged off and told so (AC-11, AC-12, V-222389, V-222390).
- **SEC-22** SHALL: The session identifier is replaced at logon and whenever privileges change, and the old one is destroyed on the server (V-222443).
- **SEC-23** SHALL: Every screen has a *Log off* control. Logging off destroys the server session, clears cookies and client storage, and shows a logoff message (V-222388, V-222391, V-222392).
- **SEC-24** SHALL: The number of concurrent sessions per user is limited to the value in the system profile (AC-10, V-222387).
- **SEC-25** SHALL: Every request that changes state carries protection against cross-site request forgery, and the server checks `Origin`.

## Access control

- **SEC-30** SHALL: Access is denied unless a rule allows it. Every route declares the role or attribute it needs in the route table, and a test fails for a route that declares none (AC-3).
- **SEC-31** SHALL: Authorization is enforced on the server for every request, on the object as well as the route: a user who may read *a* record is checked for *this* record. Hiding a control in the UI is not access control.
- **SEC-32** SHALL: Roles follow least privilege and separation of duties. Administration, audit review and ordinary use are separate roles, and no role can both perform an action and erase its audit record (AC-5, AC-6).
- **SEC-33** SHALL: A privileged function re-checks the user's credential or requires a recent logon.

## Input and output

- **SEC-40** SHALL: All input is validated on the server, at the boundary, against an allowlist: type, length, range, format and permitted values. Input that fails is rejected, not repaired. This covers bodies, query strings, headers, path parameters, file names and uploaded content (SI-10).
- **SEC-41** SHALL: Request bodies are size-limited, and unknown JSON fields are rejected (`DisallowUnknownFields`).
- **SEC-42** SHALL: SQL uses placeholders only. Commands are run with an argument list, never a shell string. Input is never used to build a file path without cleaning it and confirming it stays inside its root.
- **SEC-43** SHALL: Output is encoded for its context. Server-rendered HTML uses `html/template`, never `text/template`. In the browser, user, repository and agent content is sanitised before it is inserted (see the React frontend standard, FE-60).
- **SEC-44** SHALL: Responses carry `Content-Security-Policy` (no `unsafe-inline` or `unsafe-eval` for scripts), `Strict-Transport-Security`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: no-referrer` and `frame-ancestors 'none'` unless framing is required. Responses with restricted data carry `Cache-Control: no-store`.
- **SEC-45** SHALL: An uploaded file is checked for type by content, size-limited, stored outside the web root under a generated name, and never executed or served back with an executable content type.
- **SEC-46** SHALL: An error shown to a user says what to do and gives a reference number. It never contains a stack trace, a query, a path, a version or a secret. The detail goes to the log (SI-11).

## Cryptography and data protection

- **SEC-50** SHALL: All cryptography is done by the FIPS 140-3 validated module (see SSC-01). The application refuses to start when `crypto/fips140.Enabled()` is false (SC-13).
- **SEC-51** SHALL: No other cryptographic library is linked, and no algorithm is implemented in project code. MD5, SHA-1, DES, 3DES and RC4 are not used for any security purpose.
- **SEC-52** SHALL: All traffic uses TLS 1.2 or later, with `MinVersion` set explicitly. There is no plain-HTTP listener other than a redirect. Outbound calls verify the server certificate, and `InsecureSkipVerify` never appears in the code (SC-8).
- **SEC-53** SHALL: Server certificates come from the DoD PKI, or from the authority the AO approves. Self-signed certificates are for local development only.
- **SEC-54** SHALL: Restricted data is encrypted at rest with a FIPS-validated mechanism: the platform's validated volume or database encryption, or field encryption done with the validated module. The data-flow table says which (SC-28).
- **SEC-55** SHALL: Keys and secrets come from the platform's secret store at run time. They are never in the repository, the image, a build argument, an environment file in the repository, a log or an error. Keys can be rotated without a new build (SC-12).
- **SEC-56** SHALL: Random values used for security come from `crypto/rand`. `math/rand` is not used for identifiers, tokens or keys.

## Resilience

- **SEC-60** SHALL: The application fails closed. When it cannot check a credential, a permission or a revocation, or cannot write the audit record, the request is refused (SC-24, AUD-40).
- **SEC-61** SHALL: Every listener sets timeouts for headers, reads, writes and idle connections, and limits request rate per user and per source.
- **SEC-62** SHALL: The application runs as a non-root user with a read-only file system, apart from the data directories it declares, and needs no added operating-system capabilities (CM-7).
- **SEC-63** SHALL: Debug endpoints, profilers, sample content, default accounts and directory listings are not present in a release build.

## Why

- **These are the findings assessors look for.** The ASD STIG is the checklist an assessor will run. Building to it, with a test per rule, turns the assessment into a review of evidence.
- **Identity comes from the Department.** Using CAC, PIV and the approved identity provider means the application inherits account vetting and holds no passwords to lose.
- **Fail closed.** A system that keeps working when it cannot check a permission or record an action has no control at all at the moment one is needed.
