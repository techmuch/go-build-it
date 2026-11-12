# Code design

How code is shaped so that people and AI members can understand it, test it and change it safely: deep modules, a small number of strong abstractions, and tooling that refuses bad choices. Part of the engineering-practice set, with *Planning and alignment* (`PLAN-`) and *Development loop* (`DEV-`).

- Every rule has an ID. Review findings and project overrides cite it (`DES-04`).
- **SHALL** rules block a merge. **SHOULD** rules may be skipped with a reason in the pull request.
- Draws on Matt Pocock's engineering skills for agents, John Ousterhout (deep modules), Michael Feathers (seams) and Martin Fowler (code smells).

## Words used here

- A **module** is anything with an interface and an implementation: a function, a type, a package.
- Its **interface** is everything a caller must know to use it correctly: the signatures, and also the invariants, the order of calls, the errors, the configuration and the performance.
- **Depth** is how much behaviour a caller gets for each thing they must learn. A **deep** module has a small interface over a lot of behaviour. A **shallow** one has an interface nearly as complicated as what it does.
- A **seam** is a place where behaviour can be changed without editing the code there. An **adapter** is one concrete thing plugged into a seam.

## Deep modules

- **DES-01** SHALL: Modules are deep. A module hides a real amount of complexity behind an interface that is small and simple to state.
- **DES-02** SHALL: Before adding to an interface, ask whether the number of functions can be reduced, the parameters simplified, or more complexity hidden inside. Add to the interface only when the answer to all three is no.
- **DES-03** SHALL: A module passes the **deletion test**: if it were deleted, its complexity would reappear in several callers. A module whose removal makes complexity vanish is a pass-through, and is removed.
- **DES-04** SHALL: Callers and tests use the same interface. When a test needs to reach past the interface to check something, the module has the wrong shape, and the shape is fixed.
- **DES-05** SHALL: A module owns its data and its rules. Other code asks it to do things. It does not reach into the module's data and decide for it.
- **DES-06** SHALL: An interface is hard to misuse. Required values are parameters, invalid states cannot be built, and the order of calls is enforced by the types where the language allows.
- **DES-07** SHALL: Errors are part of the interface. A module defines the few errors a caller can act on, and handles the rest inside.
- **DES-08** SHOULD: What a module does can be said in one sentence without "and". When it cannot, it is two modules.

## Seams and abstractions

- **DES-10** SHALL: A seam is added where something really varies: there are two adapters, or one adapter and a test that needs another. One adapter and no such test is a hypothetical seam, and is not built.
- **DES-11** SHALL: Seams sit at the edges of the system (the network, the clock, the file system, other programs, paid services). Inside, code calls code directly.
- **DES-12** SHALL: The project has a small number of strong abstractions that most work goes through (one way to declare an operation, one way to load data, one way to report an error). A new feature uses them. It does not invent a parallel way.
- **DES-13** SHALL: An abstraction is introduced when it removes a decision from every later change, and it is recorded in a decision record with the one right way to use it.
- **DES-14** SHALL: Nothing is built for a need that does not exist yet: no unused parameter, option, hook, layer or configuration. It is added when the second real case arrives.
- **DES-15** SHALL: Duplication is removed when the copies are the same idea. Code that merely looks alike and changes for different reasons stays separate.

## Tooling that narrows the choices

- **DES-20** SHALL: The strictest practical settings of the compiler, type checker and linter are on from the first commit. A rule is switched off on one line with a reason, never for the project.
- **DES-21** SHALL: Types carry domain meaning. A domain concept from the glossary has its own type, and is not passed around as a bare string or number. There is no `any`, no unchecked cast and no ignored error.
- **DES-22** SHALL: Architecture rules are enforced by tools, not by memory: which package may import which, where input and output may happen, which module owns a table. A violation fails the build.
- **DES-23** SHALL: When a review finds the same kind of mistake twice, a lint rule, a type or a test is added so it cannot happen a third time. Guidance in a document is the last resort.
- **DES-24** SHALL: One command runs every check a reviewer would run, and it is fast enough to run after every small change. An AI member runs it before saying work is done.
- **DES-25** SHALL: Formatting is done by a tool and never discussed in review.

## Finding your way

- **DES-30** SHALL: Code is found by name. Directory, file, type and function names use the glossary's terms and say what the thing is, so a search for a domain term finds its code.
- **DES-31** SHALL: Each module opens with a short comment: what it is for, what it hides, and what a caller must know. An AI member can then decide whether to read further without reading everything.
- **DES-32** SHALL: Things that change together live together. One logical change that needs edits scattered over many files is a sign to gather them.
- **DES-33** SHALL: Comments explain why: the constraint, the decision, the trap. Code that needs a comment to explain what it does is rewritten.
- **DES-34** SHALL: Dead code, commented-out code and unused options are deleted. Version control remembers them.

## Smells

- **DES-40** SHALL: A reviewer looks for these in every change and names any it finds. Each is a judgement call to discuss, not an automatic failure, and a rule in a project standard overrides it:
  - *Mysterious name*: the name does not say what the thing does or holds.
  - *Duplicated code*: the same logic appears more than once in the change.
  - *Feature envy*: a function uses another module's data more than its own.
  - *Data clumps*: the same few values always travel together.
  - *Primitive obsession*: a bare string or number stands for a domain concept.
  - *Repeated switches*: the same branching on the same type in several places.
  - *Shotgun surgery*: one change, many scattered edits.
  - *Divergent change*: one module edited for unrelated reasons.
  - *Speculative generality*: structure for needs nobody has.
  - *Message chains*: `a.b().c().d()` that callers should not depend on.
  - *Middle man*: a module that only passes calls on.
- **DES-41** SHALL: A smell that tooling could catch becomes a tool rule (DES-23).

## Keeping it healthy

- **DES-50** SHALL: Each task leaves the code it touched no worse, and tidies what it had to understand. Larger restructuring is its own task, separate from behaviour changes.
- **DES-51** SHALL: Restructuring is done in small steps, each of which keeps the tests passing, and changes no behaviour.
- **DES-52** SHOULD: The project regularly looks for shallow modules and scattered knowledge to deepen. The work is planned like any other, with a written plan of small steps before it starts.

## Why

- **Small interfaces fit in a context window.** An agent can hold a deep module's interface in mind and change what is behind it safely. A tangle of shallow modules must all be read to change one.
- **Fewer choices, better choices.** Strong abstractions and strict tooling leave an agent few ways to do a thing, and those ways are the good ones. Freedom is where the mud comes from.
- **Tools do not forget.** A rule in a linter is applied every time. A rule in a document is applied when someone remembers it.
- **The test surface is the interface.** Code that is easy to test through its interface is, for the same reason, easy to use and easy to change.
