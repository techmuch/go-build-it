# DoD ATO readiness

How a project is run so that its software can receive a Department of Defense Authorization to Operate (ATO) under the NIST Risk Management Framework (RMF).

An ATO is granted by an Authorizing Official (AO) to a *system in its environment*, not to source code. This standard cannot grant one. It makes the software, and the evidence about it, ready for assessment. Three more standards carry the technical rules: *DoD application security* (`SEC-`), *DoD audit logging* (`AUD-`) and *DoD secure build and supply chain* (`SSC-`).

- Every rule has an ID. Review findings and overrides cite it (`ATO-12`).
- **SHALL** rules block a merge. A SHALL rule in the four DoD standards can be overridden only with the written agreement of the project's ISSM or AO, recorded in an ADR.
- Where a DoD standard and another standard disagree, the DoD standard wins (ATO-40).

## System profile

- **ATO-01** SHALL: `docs/ato/system-profile.md` exists before any code is written. It states: the mission and owner; the DoD Impact Level (IL2, IL4, IL5 or IL6); the kinds of data handled (public, CUI by category, PII, PHI, classified); the RMF categorization (confidentiality, integrity, availability, each low, moderate or high); the hosting platform; and the AO, ISSM and ISSO by role.
- **ATO-02** SHALL: Until the profile says otherwise, the project is built for **IL5 with CUI at moderate/moderate/moderate**. An unknown is treated as the stricter case.
- **ATO-03** SHALL: A project profiled for IL6 or for classified data stops and asks the AO for its additional requirements before design. This standard does not cover classified development.
- **ATO-04** SHALL: The profile names the hosting platform and its authorization (for example a DoD provisional authorization at the target Impact Level). Software is never deployed to a platform authorized below the profile's level.

## Controls and inheritance

- **ATO-10** SHALL: The control baseline is NIST SP 800-53 Rev. 5 with the DoD overlays the AO names. `docs/ato/controls.md` lists every control in the baseline as one of: *inherited* (from whom), *shared* (who does which part), *implemented* (where), or *not applicable* (why).
- **ATO-11** SHALL: An *implemented* control has an implementation statement that names the code, configuration or test that satisfies it. A requirement that implements a control names the control ID (`AC-7`), and the code names the requirement ID, so a control traces to code and to a test.
- **ATO-12** SHALL: Inherited controls are confirmed against the platform's own customer responsibility matrix. A control is not marked inherited on assumption.
- **ATO-13** SHALL: The applicable DISA STIGs and SRGs are listed in the profile, with the release used. The Application Security and Development STIG always applies. A STIG checklist (`.ckl` or `.cklb`) is kept current in `docs/ato/stig/`, and every *open* or *not applicable* finding has a reason.
- **ATO-14** SHALL: When DISA publishes a new release of an applicable STIG, a task is opened to assess the differences within 30 days.

## Boundary and data flow

- **ATO-20** SHALL: `docs/ato/boundary.html` shows the authorization boundary: every component, every data store, every external system, every connection across the boundary, and which side of it each person and AI member works on.
- **ATO-21** SHALL: `docs/ato/ports-protocols.md` lists every port, protocol and service the software listens on or calls, with its purpose and direction. The software opens nothing that is not on the list, and a test checks the listeners against it.
- **ATO-22** SHALL: A data-flow table lists each kind of data, where it enters, where it is stored, where it leaves, and how it is protected in transit and at rest.
- **ATO-23** SHALL: A change that adds a component, a data store, an external connection or a port updates these documents in the same pull request.

## Evidence

- **ATO-30** SHALL: Evidence is produced by the pipeline, not written by hand: test results, static-analysis and vulnerability reports, the SBOM, the signed build record, and the STIG checklist. Each release keeps its own set under a release tag.
- **ATO-31** SHALL: A weakness that is not fixed before release is recorded in `docs/ato/poam.md` (the Plan of Action and Milestones) with: what it is, the control or STIG rule it affects, its severity, the mitigation in place, the owner and the due date.
- **ATO-32** SHALL: Nothing is ever removed from the POA&M. An item is closed with the evidence that closed it.
- **ATO-33** SHALL: Every release has a short security impact statement: what changed, which controls it touches, and whether the boundary, the data handled or the ports changed. A change to any of those three needs the ISSM's approval before merge.
- **ATO-34** SHOULD: Control statements are also kept in OSCAL, so they can be loaded into the AO's governance tool (for example eMASS) without retyping.

## Precedence over the other standards

- **ATO-40** SHALL: These rules replace their counterparts in other standards:
  - Requests from the local machine are **not** trusted. Every request is authenticated (SEC-01).
  - The software does not update itself. It is deployed only by the authorized pipeline (SSC-41).
  - Cryptography uses the FIPS-validated module only (SSC-01).
  - The audit log (AUD) is kept in addition to the product history. Neither replaces the other.
- **ATO-41** SHALL: The pure-Go, single-binary and no-third-party-origin rules of the Go backend and React frontend standards stay in force. They are part of the security case.

## People and AI members

- **ATO-50** SHALL: No CUI, PII, PHI, classified data, real credential or real mission data is ever placed in the repository, a test fixture, a screenshot, a prompt, an agent transcript, a task, a question or the history. Tests use synthetic data that is labelled as synthetic.
- **ATO-51** SHALL: An AI member works only with material its harness and model service are authorized to receive. The profile records each harness, where it sends data, and the highest data level it may see. When that is unknown, the answer is: public and synthetic data only.
- **ATO-52** SHALL: AI members have no access to production systems, production data or production credentials.
- **ATO-53** SHALL: Every change is reviewed and approved by a named person who did not write it, before merge. An AI member's review is advice. It is never the approval.
- **ATO-54** SHALL: The author, the reviewer and the approver of every change, human or AI, are recorded and can be reported for any release.
- **ATO-55** SHALL: Anyone, person or AI member, who finds that restricted data has entered the repository or a transcript stops work and reports it to the ISSO at once. Deleting it quietly is not a fix.

## Why

- **The AO decides on evidence.** A control that is implemented but cannot be shown is, for an assessor, not implemented. Producing the evidence from the pipeline keeps it true.
- **Most controls are inherited.** Stating the Impact Level and the platform first shows which controls the software must implement itself, which is usually a small share.
- **Agents change the risk.** AI members read everything they are given and send it to a model service. Keeping restricted data out of their reach, and keeping a person as the approver, is what lets a team use them inside this process.
