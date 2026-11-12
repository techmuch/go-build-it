# AI layer security

How a project uses AI models safely inside the product: models that read content, produce text or code, or call tools. Part of the hardening set, with *Software hardening* (`HARD-`) and *Operational resilience* (`OPS-`).

- Every rule has an ID. Review findings and project overrides cite it (`AIS-12`).
- **SHALL** rules block a merge. **SHOULD** rules may be skipped with a reason in the pull request.
- The starting point: **a model's output is untrusted input, and anything a model reads can instruct it.** No filter reliably stops prompt injection, so the design must stay safe when injection succeeds.

## Where AI is used

- **AIS-01** SHALL: `docs/security/ai-inventory.md` lists every use of a model: its purpose, the model and provider, what data it is sent, what it can read, which tools it can call, and what its output can change.
- **AIS-02** SHALL: Each use states what happens if the model is wrong, manipulated or unavailable. A feature must have a safe result in all three cases.
- **AIS-03** SHALL: Security decisions are never made by a model. Authentication, authorization, input validation and the choice of what to audit are deterministic code.

## What a model reads

- **AIS-10** SHALL: Everything a model reads that the project did not write is untrusted: user text, documents, web pages, repository files, tool results, e-mail and other models' output. It is marked as data in the prompt and kept apart from the project's instructions.
- **AIS-11** SHALL: Instructions found inside untrusted content are not followed. The system prompt says so, and a test with injected instructions in each kind of content proves the feature still behaves.
- **AIS-12** SHALL: Injection filtering and classifiers may be added as one layer. They are never the only protection, and the threat model does not count them as a control.
- **AIS-13** SHALL: A model is sent only the data its task needs. Secrets, credentials and other users' data are never placed in a prompt.
- **AIS-14** SHALL: The data level each model provider may receive is recorded in the inventory. Data above that level is never sent, and the code enforces this where the request is built.

## What a model can do

- **AIS-20** SHALL: A model acts only through declared tools. Each tool has typed arguments that are validated like any other input, and each call is checked against the permissions of the principal the model acts for, by the same access-control check as every other request.
- **AIS-21** SHALL: An AI member or agent has its own identity and the smallest set of tools and permissions for its task. It never holds a person's credentials.
- **AIS-22** SHALL: These three are never combined in one unreviewed step: reading untrusted content, access to private data, and a way to send data out or change something. Where a task needs all three, a person approves the action in between.
- **AIS-23** SHALL: Actions that cannot be undone, cost money or are visible outside the system (sending, publishing, deleting, paying, deploying, merging) need a person's approval, shown with exactly what will happen.
- **AIS-24** SHALL: Code or commands produced by a model run only in a sandbox with no credentials, a limited file system and declared network access.
- **AIS-25** SHALL: Every agent run has limits on steps, time, tokens and spend. Reaching a limit stops the run and reports it.

## What a model produces

- **AIS-30** SHALL: Model output is validated by deterministic code before anything uses it. Structured output is parsed against a schema and rejected when it does not fit. It is never passed to `eval`, a shell, a SQL string or a template unescaped.
- **AIS-31** SHALL: Model output shown to a person is sanitised like any user content. Links and images in it cannot load from or send data to another origin without the user choosing to.
- **AIS-32** SHALL: Code written by a model goes through the same pipeline as code written by a person: tests, static analysis, security scans and review by a person.
- **AIS-33** SHALL: A dependency suggested by a model is checked to exist, to be the intended package and to be maintained before it is added. It follows the same approval as any dependency.
- **AIS-34** SHALL: People can always tell what a model produced. AI-generated content and AI-taken actions are labelled in the UI and recorded under the AI member's own identity.
- **AIS-35** SHALL: Where the product states facts from a model, it shows the source the fact came from, or says that it is unverified.

## Models, prompts and data

- **AIS-40** SHALL: The model and its version are pinned in configuration. Changing either is a reviewed change that reruns the evaluations (AIS-50).
- **AIS-41** SHALL: Prompts are files in the repository, versioned and reviewed like code. No prompt is assembled from strings scattered through the program.
- **AIS-42** SHALL: Model files that the project hosts are fetched from a recorded source, verified by checksum and listed in the SBOM. Formats that can run code when loaded are not used.
- **AIS-43** SHALL: For any model the project trains or fine-tunes, every dataset has a recorded source, licence, version, checksum and the person who approved it. A model can be traced to the exact data and code that produced it.
- **AIS-44** SHALL: Content retrieved for a model (search indexes, vector stores, memory) keeps the access rules of its source. A user's question never retrieves a document that user could not open.
- **AIS-45** SHALL: Anything written into long-lived memory or an index by a model, or from untrusted content, is recorded with its origin and can be found and removed.

## Evaluation and records

- **AIS-50** SHALL: Each AI feature has an evaluation set in the repository: normal cases, edge cases and attack cases (injection, data extraction, tool misuse). It runs in the pipeline, and a drop below the agreed score blocks the change.
- **AIS-51** SHALL: Every injection or misuse found in testing or production is added to the evaluation set before it is fixed.
- **AIS-52** SHALL: Every model call records the feature, the model and version, the prompt version, the principal, the tools called with their arguments and results, the tokens and cost, and the outcome. Restricted data is kept out of this record by the same rules as any log.
- **AIS-53** SHALL: Unusual behaviour raises an alert: a jump in tool calls, denials, spend or refused outputs for one principal or feature.
- **AIS-54** SHALL: Each AI feature can be switched off by configuration without a release, and the product still works without it.

## Why

- **Injection will get through.** Models cannot reliably tell instructions from data. Safety has to come from what the model is able to reach and do, not from hoping it refuses.
- **Output is input.** Treating what a model produces like text from an unknown user puts it behind protections the project already has.
- **A person decides what matters.** Approval at the points that cannot be undone keeps people accountable while agents do the work in between.
- **Change is constant.** Models and prompts change behaviour without a code change. Pinning them and running evaluations is how a team notices.
