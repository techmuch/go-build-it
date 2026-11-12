# Development loop

How code gets written and checked: test first, in small steps, with fast feedback, by one AI member, and reviewed by another with a fresh context. Part of the engineering-practice set, with *Planning and alignment* (`PLAN-`) and *Code design* (`DES-`).

- Every rule has an ID. Review findings and project overrides cite it (`DEV-11`).
- **SHALL** rules block a merge. **SHOULD** rules may be skipped with a reason in the pull request.
- Draws on Matt Pocock's engineering skills for agents and Kent Beck (test-driven development).

## Test first, one slice at a time

- **DEV-01** SHALL: Behaviour is built test first. Write one test for one behaviour and watch it fail for the right reason. Write the least code that passes it. Then tidy with the tests passing. Repeat.
- **DEV-02** SHALL: Tests and code are written in turn, never in bulk. Writing all the tests and then all the code is not test-driven development: tests written ahead describe imagined behaviour.
- **DEV-03** SHALL: Before the first test, the AI member lists the behaviours to be tested, in the glossary's words and in priority order, and agrees the public interface. The list comes from the task's acceptance checks.
- **DEV-04** SHALL: The first test is a tracer bullet: one path through every layer the feature needs. It proves the pieces connect before any of them is built out.
- **DEV-05** SHALL: A test that passes the first time it is run has proved nothing. It is broken on purpose once, by breaking the code, to see it fail.
- **DEV-06** SHALL: Tidying happens only when the tests pass, changes no behaviour, and adds no test.

## What a good test is

- **DEV-10** SHALL: A test checks behaviour through the module's public interface. It does not call private functions, inspect internal state, or look in the database behind the interface's back.
- **DEV-11** SHALL: A test survives a refactoring. If renaming an internal function or moving code breaks a test while behaviour is unchanged, the test is wrong and is rewritten.
- **DEV-12** SHALL: The expected value comes from an independent source: a literal, a worked example or the requirement. It is never computed the way the code computes it, because such a test cannot disagree with the code.
- **DEV-13** SHALL: A test's name states a capability in the glossary's words ("a reviewer can approve a change they did not write"), so the list of test names reads as a specification.
- **DEV-14** SHALL: Only the edges of the system are replaced in tests: the network, the clock, paid services, other programs. A project's own modules are used for real. An internal collaborator is not mocked.
- **DEV-15** SHALL: A fake that stands in for something real is checked against the real thing by a test that can be run on demand.
- **DEV-16** SHALL: Tests are fast and give the same result every time. A test that fails now and then is fixed or deleted the day it is noticed. It is never retried until it passes.
- **DEV-17** SHALL: Testing effort goes where being wrong is costly: the main paths, the rules of the domain and the complicated logic. Not every line needs a test, and every important behaviour does.

## Feedback before fixes

- **DEV-20** SHALL: The feedback loop is fast: the type check and the tests for the code being changed run in seconds, and the AI member runs them after every small step.
- **DEV-21** SHALL: A bug is fixed in this order: build a way to make it happen on demand, reduce that to the smallest case, form a hypothesis, test it, fix it, and keep the reproduction as a regression test.
- **DEV-22** SHALL: No fix is attempted before the bug can be reproduced. A change made on a guess, with no failing test, is not a fix.
- **DEV-23** SHALL: An AI member that has tried the same approach twice without progress stops and reports what it tried, what it saw and what it suspects. It does not try a third time or weaken a test.
- **DEV-24** SHALL: A failing check is never made to pass by deleting the test, loosening the assertion, skipping it or silencing the linter. If the check is wrong, that is said in the pull request and a person agrees.
- **DEV-25** SHALL: "Done" means the checks were run and passed, and the report says which were run. Work that was not verified is reported as not verified.

## Small steps

- **DEV-30** SHALL: A commit is one logical change that leaves every check passing. Its message says why.
- **DEV-31** SHALL: A change to behaviour and a change to structure are separate commits, and where they are large, separate pull requests.
- **DEV-32** SHALL: A pull request does one thing and is small enough to review properly in one sitting. A change too large for that is cut into slices that each stand alone.
- **DEV-33** SHALL: An AI member changes only what its task needs. Something else worth fixing becomes a new task, not an extra in this change.

## Implementer and reviewer

- **DEV-40** SHALL: The AI member that writes a change does not review it. Review is done by a different AI member that starts with a fresh context.
- **DEV-41** SHALL: The reviewer is given the diff, the task, the requirements it serves, the standards and the glossary. It is not given the implementer's conversation, so it judges the work, not the story of the work.
- **DEV-42** SHALL: The review has two separate parts. **Standards**: does the change follow the project's standards? Each finding cites the rule ID. **Specification**: does the change do what the task and requirements asked, all of it and nothing else? Each finding cites the requirement.
- **DEV-43** SHALL: The two parts are assessed separately, so that a change which is well written and wrong, or right and badly written, is reported as exactly that.
- **DEV-44** SHALL: A finding says where, what is wrong, which rule or requirement it breaks, and whether it is a firm violation or a judgement call. A reviewer skips anything the tooling already enforces.
- **DEV-45** SHALL: The reviewer reports and does not fix. Fixes go back to an implementer, and the result is reviewed again.
- **DEV-46** SHALL: The reviewer runs the checks itself. It does not accept the implementer's statement that they pass.
- **DEV-47** SHALL: A person gives the approval. The AI review is evidence for that person, set out so they can check its claims quickly. It is never the approval.
- **DEV-48** SHALL: Review budget is planned. A task's estimate includes its review, and review is not skipped to save time or tokens.

## Context is a budget

- **DEV-50** SHALL: An AI member's context is used for the task in hand. It reads what it needs and delegates wide searches and other exploration that would fill its context with material it will not use.
- **DEV-51** SHALL: When a context is nearly full, the AI member stops at a safe point and writes a handoff: the objective and what done means, the current state, the decisions made and why, the dead ends, the next steps, and the files that matter.
- **DEV-52** SHALL: A handoff is written for a reader who knows nothing of the session, and the next AI member starts from it and the repository, not from the old conversation.
- **DEV-53** SHALL: What must outlast a session is written to the repository: decisions, glossary terms, plans and findings. A session's memory is not a place to keep anything.

## Skills, not a framework

- **DEV-60** SHALL: Repeated ways of working are written as small **skills**: one Markdown file each, for one job, that a person can read in a few minutes and change. They live in the repository and are reviewed like code.
- **DEV-61** SHALL: Skills are composed by a person or an AI member choosing the next one. No fixed pipeline decides the order of work or hides its steps.
- **DEV-62** SHALL: A skill says when to use it, what it needs, what it produces and when to stop and ask. It has no hidden behaviour: what it tells the AI member to do is all in the file.
- **DEV-63** SHALL: At every step a person can see what an AI member is doing and why, can stop it, and can take over. Automation that removes any of these is not adopted.
- **DEV-64** SHALL: A skill that keeps producing the same mistake is changed, and the change is recorded with the reason. Ways of working are improved from evidence, like code.

## Why

- **Small verified steps beat one big attempt.** An agent that writes a feature in one go produces something plausible that nobody has checked. A failing test before each step means every step was checked by something other than the agent's confidence.
- **Fresh eyes catch more.** An implementer's context is full of its own reasoning, and it reviews its own work kindly. A reviewer that sees only the result and the rules does not.
- **Feedback is the control.** Types, tests and linters tell an agent it is wrong in seconds, without a person. The faster and stricter they are, the less a person has to catch.
- **People stay in charge.** Small, readable skills keep the way of working in the team's hands. A framework that hides its steps takes the design away from the people who answer for it.
