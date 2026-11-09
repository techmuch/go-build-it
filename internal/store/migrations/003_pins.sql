-- Viewer pins: comments placed on a spot of an attachment or mockup (FR-110, FR-102a).
CREATE TABLE pins (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  subject    TEXT NOT NULL,          -- what the viewer was opened for, e.g. Q-34, a review id or a project id
  item       TEXT NOT NULL,          -- the attachment: its URL, or its name when it has none
  project_id TEXT NOT NULL DEFAULT '',
  x          REAL NOT NULL,          -- percent of the item's width
  y          REAL NOT NULL,          -- percent of the item's height
  text       TEXT NOT NULL,
  author     TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE INDEX pins_subject_item ON pins(subject, item);
