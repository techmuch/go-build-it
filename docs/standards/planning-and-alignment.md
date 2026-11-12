# Planning and alignment

How people and AI members reach a shared understanding before code is written, and keep it. The aim is that the person, the agent and the codebase mean the same thing by the same words. Part of the engineering-practice set, with *Code design* (`DES-`) and *Development loop* (`DEV-`).

- Every rule has an ID. Review findings and project overrides cite it (`PLAN-12`).
- **SHALL** rules block a merge. **SHOULD** rules may be skipped with a reason in the pull request.
- Draws on Matt Pocock's engineering skills for agents, Eric Evans (ubiquitous language) and Michael Nygard (decision records).

## The interview comes first

- **PLAN-01** SHALL: Work that changes behaviour or design starts with an interview, not with code. The AI member questions the person until both could describe the result the same way: what it does, what it does not do, and how anyone will know it is done.
- **PLAN-02** SHALL: Questions are asked one at a time, and each comes with the AI member's recommended answer and its reason. The person can accept it with one word.
- **PLAN-03** SHALL: A question the repository can answer is answered by reading the repository. The person is asked only what the code, the documents and the history cannot say.
- **PLAN-04** SHALL: The interview walks every branch of the design: inputs, outputs, errors, limits, who may do it, what happens to existing data, and what is out of scope. A decision that depends on another is asked after it.
- **PLAN-05** SHALL: When what the person says disagrees with the code, the glossary or an earlier decision, the AI member says so at once and asks which is right. It never picks silently.
- **PLAN-06** SHALL: Each answer is recorded where it will be kept as soon as it is given: a requirement, a glossary term or a decision record. Nothing important lives only in a conversation.
- **PLAN-07** SHALL: The interview ends when no open question would change the design. The AI member then writes a summary of what was agreed, and the person approves it before implementation starts.
- **PLAN-08** SHALL: Small, well-understood work skips the interview. That means a fix with a failing test, or a change whose result is fully stated in its task. When in doubt, ask one question: "what would make this wrong?"

## One language

- **PLAN-10** SHALL: The project has a glossary, `CONTEXT.md` at the repository root. Each entry is a term, a one- or two-sentence definition, and the words to avoid for it ("Customer. Avoid: client, account").
- **PLAN-11** SHALL: The glossary holds domain language only. It contains no implementation detail, no plan and no decision.
- **PLAN-12** SHALL: The glossary's terms are the only names for those things in requirements, tasks, code identifiers, database columns, API fields, UI text and tests. A second name for one thing, or one name for two things, is a defect.
- **PLAN-13** SHALL: A new domain term is added to the glossary in the pull request that first uses it. Renaming a term renames it everywhere in the same change.
- **PLAN-14** SHALL: Vague or overloaded words are challenged when they appear ("by *account* do you mean the Customer or the User?"), and the answer goes into the glossary.
- **PLAN-15** SHALL: Relationships between terms are tested with concrete scenarios, including awkward ones ("an order is cancelled after half of it shipped"), before they are relied on.
- **PLAN-16** SHALL: Every AI member reads the glossary before it plans, writes or reviews. An invented term that is not in the glossary is a review finding.
- **PLAN-17** SHOULD: A repository with several distinct areas has one glossary per area, and a `CONTEXT-MAP.md` that says where each is and how the areas relate.

## Decisions

- **PLAN-20** SHALL: A decision gets a record (an ADR in `docs/decisions/`) when it is hard to reverse, surprising to a newcomer, and the result of a real choice between alternatives. Routine choices do not.
- **PLAN-21** SHALL: A decision record states the context, the decision, the alternatives that were turned down and why, and the consequences. It is short, and it is never edited to change its meaning: a new record supersedes it.
- **PLAN-22** SHALL: Hard-to-reverse decisions are made by a person. An AI member lays out the options with a recommendation, and the person chooses.
- **PLAN-23** SHALL: For an interface that will be hard to change, at least two noticeably different designs are produced and compared before one is chosen. The comparison goes in the decision record.
- **PLAN-24** SHALL: A question that argument cannot settle (will this state model feel right, will this layout work) is answered with a throwaway prototype. The prototype is deleted, and what it taught is written down.

## From plan to work

- **PLAN-30** SHALL: A feature is written up before it is built: the problem, who has it, what will be true when it is solved, what is out of scope, and how it will be checked. Requirements get stable IDs.
- **PLAN-31** SHALL: Work is cut into **vertical slices**. Each slice goes through every layer it needs, from the interface to storage, and can be demonstrated and tested by itself. Work is never split into "all the database, then all the API, then all the UI".
- **PLAN-32** SHALL: The first slice is a tracer bullet: the thinnest path that proves the pieces connect. Later slices widen it.
- **PLAN-33** SHALL: A task states the behaviour wanted, its acceptance checks, the requirements it serves, what it depends on, and what it must not touch. It does not prescribe the code.
- **PLAN-34** SHALL: A task is small enough for one AI member to finish in one context window with room to spare. A task that needed a compaction or a handoff is a signal to cut the next ones smaller.
- **PLAN-35** SHALL: Each task says whether it can run without a person or needs one, and for what. Tasks that need a person are marked before work starts, not found out halfway.
- **PLAN-36** SHALL: When work shows the plan was wrong, the plan is changed first, with the reason, and then the work continues. Code and plan do not drift apart.

## Why

- **Shared understanding is the bottleneck.** An agent can write code faster than a person can read it. What it cannot do is know what the person meant. The interview is the cheapest place to find out.
- **Words steer agents.** An agent that reads "account" in one file and "customer" in another builds two things. One glossary removes a whole class of wrong work.
- **Thin slices show the truth early.** A slice that runs end to end finds the wrong assumption in an hour. Layers built one at a time hide it until the end.
- **People own the design.** Agents propose and build. The decisions that are costly to undo stay with someone who will answer for them.
