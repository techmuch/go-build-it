-- Learn: lesson progress per user (FR-134, FR-135, ADR-012). A step is done when its
-- history event appears in the practice project after the lesson started, when its view
-- was visited, or when the user ticked it. The last two are stored here.
CREATE TABLE learn_started (
  user       TEXT NOT NULL,
  lesson     TEXT NOT NULL,
  started_at TEXT NOT NULL,
  PRIMARY KEY (user, lesson)
);
CREATE TABLE learn_steps (
  user   TEXT NOT NULL,
  lesson TEXT NOT NULL,
  step   INTEGER NOT NULL,
  how    TEXT NOT NULL,          -- tick | visit
  at     TEXT NOT NULL,
  PRIMARY KEY (user, lesson, step)
);
