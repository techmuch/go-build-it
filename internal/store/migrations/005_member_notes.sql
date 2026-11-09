-- FR-76: a member's lasting knowledge lives in go-build-it, not only in its conversation.
CREATE TABLE member_notes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  member_id TEXT NOT NULL,
  task_id TEXT NOT NULL,
  note TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE INDEX member_notes_member ON member_notes(member_id);
ALTER TABLE sessions ADD COLUMN pane_id TEXT NOT NULL DEFAULT '';
ALTER TABLE sessions ADD COLUMN conversation_id TEXT NOT NULL DEFAULT '';
