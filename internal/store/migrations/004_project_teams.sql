-- Rev 3 (ADR-010): global persona archetypes, per-project team members.
CREATE TABLE archetypes (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  role TEXT NOT NULL,
  instructions TEXT NOT NULL DEFAULT '',
  harness TEXT NOT NULL DEFAULT '',
  model TEXT NOT NULL DEFAULT '',
  standards TEXT NOT NULL DEFAULT '[]',
  rev INTEGER NOT NULL DEFAULT 1,
  builtin INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL DEFAULT ''
);
CREATE TABLE archetype_revisions (
  archetype_id TEXT NOT NULL,
  rev INTEGER NOT NULL,
  name TEXT NOT NULL,
  role TEXT NOT NULL,
  instructions TEXT NOT NULL,
  harness TEXT NOT NULL,
  model TEXT NOT NULL,
  standards TEXT NOT NULL,
  author TEXT NOT NULL DEFAULT '',
  note TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  PRIMARY KEY (archetype_id, rev)
);
-- Former global AI personas become archetypes, and members now belong to a project.
INSERT INTO archetypes(id,name,role,instructions,harness,model,standards,rev,builtin,updated_at)
  SELECT id,name,role,instructions,harness,model,standards,1,0,'' FROM members WHERE kind='ai';
INSERT INTO archetype_revisions(archetype_id,rev,name,role,instructions,harness,model,standards,author,note,created_at)
  SELECT id,1,name,role,instructions,harness,model,standards,'migration','from a global persona','' FROM members WHERE kind='ai';
DELETE FROM members WHERE kind='ai';
ALTER TABLE members ADD COLUMN project_id TEXT NOT NULL DEFAULT '';
ALTER TABLE members ADD COLUMN archetype_id TEXT NOT NULL DEFAULT '';
ALTER TABLE members ADD COLUMN archetype_rev INTEGER NOT NULL DEFAULT 0;
ALTER TABLE members ADD COLUMN tuning TEXT NOT NULL DEFAULT '';
ALTER TABLE members ADD COLUMN replace_instructions INTEGER NOT NULL DEFAULT 0;
ALTER TABLE members ADD COLUMN worktree TEXT NOT NULL DEFAULT '';
ALTER TABLE members ADD COLUMN pane_id TEXT NOT NULL DEFAULT '';
ALTER TABLE members ADD COLUMN conversation_id TEXT NOT NULL DEFAULT '';
ALTER TABLE members ADD COLUMN conversation_tasks INTEGER NOT NULL DEFAULT 0;
ALTER TABLE members ADD COLUMN conversation_started TEXT NOT NULL DEFAULT '';
CREATE INDEX members_project ON members(project_id);
DROP TABLE teams;
