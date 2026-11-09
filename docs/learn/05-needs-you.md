title: When the AI needs you
minutes: 7
summary: Questions instead of guesses, and members that are blocked in their terminal.
related: questions, sessions
# When the AI needs you

Members ask instead of guessing. A **blocking question** pauses its task until you answer; a non-blocking one carries an assumption that work continues on. A member can also be **blocked** in its terminal, for example by a permission prompt; you can take control of its pane from the browser.

## Step: Get a question
check: event question.asked
show: /projects/{practice}/queue [data-tour=add-task]
Add a task whose description is exactly `FAKE:ask` and run it. The scripted member asks a blocking question, just as a real one would.

## Step: Answer it
check: event question.answered
show: /questions
Open **Questions** (`g q`), read *why it matters*, pick an option and press **Answer & resume work**.

> **Pro tip:** answer from your phone: the mobile Inbox puts blocking questions first, and 🎙 lets you dictate.

## Step: Unblock a member
check: event terminal.control
show: /sessions
Add a task with the description `FAKE:block` and run it. The member stops at a permission prompt and shows as **blocked**. Open its session, switch to **Terminal**, press **⌨ Take control**, type `y` and Enter, then **Hand back**.

> **Pro tip:** watching a terminal never resizes it or takes it over; only **Take control** sends your keys, and the history records who had control and when.
