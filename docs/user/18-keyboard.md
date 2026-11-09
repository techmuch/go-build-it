section: Using go-build-it
reqs: FR-34, FR-35, FR-102a, FR-110
# Keyboard

Everything in the desktop UI can be done from the keyboard. Press <kbd>?</kbd> on any page to see this list, or use the ⌨ button next to search.

## Anywhere

| Keys | What it does |
|---|---|
| <kbd>⌘</kbd> <kbd>K</kbd> or <kbd>/</kbd> | Search and run commands. Every result shows its `gbi` CLI equivalent. |
| <kbd>?</kbd> | Show the keyboard shortcuts. |
| <kbd>Tab</kbd> / <kbd>Shift</kbd> <kbd>Tab</kbd> | Move between controls. The first <kbd>Tab</kbd> on a page offers **Skip to content**. |
| <kbd>g</kbd> then a letter | Go to a page: <kbd>p</kbd> Projects, <kbd>q</kbd> Questions, <kbd>s</kbd> Sessions, <kbd>u</kbd> Queue, <kbd>r</kbd> Reviews, <kbd>o</kbd> Options, <kbd>e</kbd> Rehearsals, <kbd>t</kbd> Team, <kbd>a</kbd> Standards, <kbd>h</kbd> History, <kbd>,</kbd> Settings, <kbd>d</kbd> Docs, <kbd>v</kbd> Change log &amp; updates. |

Single-key shortcuts are ignored while you type in a text field, so typing "g" in an answer never navigates away.

## Lists, tables and cards

Project cards, the Questions dashboard, the task queue, sessions, roadmap rows, team members and rehearsals all work the same way:

| Keys | What it does |
|---|---|
| <kbd>↑</kbd> <kbd>↓</kbd> (or <kbd>k</kbd> <kbd>j</kbd>) | Previous or next item. |
| <kbd>←</kbd> <kbd>→</kbd> | Previous or next card in the project grid. <kbd>↑</kbd> <kbd>↓</kbd> move by row there. |
| <kbd>Home</kbd> <kbd>End</kbd> | First or last item. |
| <kbd>Enter</kbd> | Open the focused item. In the task queue, <kbd>Enter</kbd> or <kbd>Space</kbd> expands and collapses a task. |

## Tabs and segmented controls

In project tabs, use <kbd>←</kbd> <kbd>→</kbd> to move and <kbd>Enter</kbd> to open. In segmented controls (such as Desktop · Tablet · Phone in the viewer), <kbd>←</kbd> <kbd>→</kbd> switch the option as you move. Buttons and link-style actions (Edit, Accept, + Milestone…) activate with <kbd>Enter</kbd> or <kbd>Space</kbd>.

## Text boxes and dialogs

| Keys | What it does |
|---|---|
| <kbd>⌘</kbd> <kbd>Enter</kbd> (<kbd>Ctrl</kbd> <kbd>Enter</kbd>) | Send, answer or save: the main button of the panel you are typing in. |
| <kbd>Esc</kbd> | Close the dialog, palette or viewer you are in. Focus goes back to where you were. |

While a dialog is open, <kbd>Tab</kbd> stays inside it.

## Attachment and mockup viewer

The strip on the left shows a thumbnail of every attachment on the item: a live preview of HTML mockups, the image itself, or the first lines of a diff, JSON file or log.

| Keys | What it does |
|---|---|
| <kbd>↑</kbd> <kbd>↓</kbd> | With the strip focused, switch to the previous or next attachment. When an item has several attachments, the strip has focus when the viewer opens. |
| <kbd>←</kbd> <kbd>→</kbd> | Previous or next attachment, from anywhere in the viewer except a text box. |
| <kbd>Esc</kbd> | Close the viewer. |

**Pinned comments.** Write the comment, turn on **📍 Pin**, then click the spot on the mockup or image. From the keyboard, <kbd>Tab</kbd> to the pin layer and press <kbd>Enter</kbd> to pin the centre. Pins are saved, shown to everyone who opens the same item, recorded in History, and can be removed with ✕. The CLI equivalents are `gbi pins list`, `gbi pins add` and `gbi pins remove`.

## Live terminal

| Keys | What it does |
|---|---|
| Every key | With control taken, goes to the agent in its herdr pane. |
| <kbd>Ctrl</kbd> <kbd>]</kbd> | Leave the terminal so page shortcuts work again. |

While you are only watching, the terminal is read-only and page shortcuts keep working.

## Mobile

The mobile UI is built for touch, but it follows the same rules with a Bluetooth keyboard: <kbd>Tab</kbd> reaches every control, <kbd>Enter</kbd> opens cards and list rows, and <kbd>↑</kbd> <kbd>↓</kbd> move through lists.
