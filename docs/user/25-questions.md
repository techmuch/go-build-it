section: How go-build-it thinks
reqs: FR-100, FR-101, FR-102, FR-138
# Questions instead of guesses

**In short:** when a member is unsure, it **asks** instead of guessing, and the question carries what you need to answer quickly.

## Why

An AI that guesses wrong early can waste hours of work. One that asks about everything wastes your time. go-build-it gives members two kinds of question:
- **Blocking:** the task pauses until you answer. Use it for decisions that change the design.
- **Non-blocking:** the member states an **assumption** and keeps working. You can confirm it or correct it later.

## What a question carries

- why it matters
- the options, with a recommendation
- attachments such as mockups (open them full screen and pin comments on them)
- the requirements involved

Blocking questions come first in the Inbox, on desktop and phone. Your answer resumes the work. An answer can also **propose a value**, so the whole team learns from it.

When a member is blocked in its *terminal* (a permission prompt, say) rather than by a question, it shows as *needs you* and you can take control of its pane from the browser.

Decisions: [ADR-009](https://github.com/techmuch/go-build-it/blob/main/docs/decisions/ADR-009-herdr-backend.md) (blocked state from herdr).
