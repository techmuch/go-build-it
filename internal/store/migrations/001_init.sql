-- go-build-it initial schema
CREATE TABLE events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  ts TEXT NOT NULL,
  actor TEXT NOT NULL,
  actor_kind TEXT NOT NULL,           -- human | persona | system
  origin TEXT NOT NULL DEFAULT 'system', -- local | tailscale | external | system
  project_id TEXT NOT NULL DEFAULT '',
  type TEXT NOT NULL,
  subject TEXT NOT NULL DEFAULT '',
  summary TEXT NOT NULL,
  why TEXT NOT NULL DEFAULT '',
  data TEXT NOT NULL DEFAULT '{}'
);
CREATE INDEX events_project ON events(project_id, id);
CREATE INDEX events_type ON events(type, id);

CREATE TABLE projects (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  path TEXT NOT NULL,
  purpose TEXT NOT NULL DEFAULT '',
  stage TEXT NOT NULL DEFAULT 'Discovery',
  forge TEXT NOT NULL DEFAULT '',
  remote TEXT NOT NULL DEFAULT '',
  website_url TEXT NOT NULL DEFAULT '',
  team_id TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE project_values (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  project_id TEXT NOT NULL,
  text TEXT NOT NULL,
  source TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'accepted', -- accepted | proposed | dismissed
  created_at TEXT NOT NULL
);

CREATE TABLE problem_defs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  project_id TEXT NOT NULL,
  text TEXT NOT NULL,
  source TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);

CREATE TABLE milestones (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL,
  ord INTEGER NOT NULL DEFAULT 0,
  name TEXT NOT NULL,
  scope TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'planned', -- planned | active | done
  gate_name TEXT NOT NULL DEFAULT '',
  gate_kind TEXT NOT NULL DEFAULT '',     -- review | tabletop | uat
  gate_status TEXT NOT NULL DEFAULT 'pending', -- pending | approved | changes
  gate_criteria TEXT NOT NULL DEFAULT '[]',
  target TEXT NOT NULL DEFAULT ''
);

CREATE TABLE teams (id TEXT PRIMARY KEY, name TEXT NOT NULL);

CREATE TABLE members (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  kind TEXT NOT NULL,          -- ai | human
  role TEXT NOT NULL,
  team_id TEXT NOT NULL DEFAULT '',
  harness TEXT NOT NULL DEFAULT '',
  model TEXT NOT NULL DEFAULT '',
  instructions TEXT NOT NULL DEFAULT '',
  login TEXT NOT NULL DEFAULT '',
  standards TEXT NOT NULL DEFAULT '[]',
  rev INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE roles (id TEXT PRIMARY KEY, name TEXT NOT NULL, description TEXT NOT NULL DEFAULT '', builtin INTEGER NOT NULL DEFAULT 0);

CREATE TABLE standards (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  body TEXT NOT NULL,
  rev INTEGER NOT NULL DEFAULT 1,
  updated_at TEXT NOT NULL
);
CREATE TABLE standard_overrides (
  standard_id TEXT NOT NULL,
  project_id TEXT NOT NULL,
  body TEXT NOT NULL,
  reason TEXT NOT NULL,
  created_at TEXT NOT NULL,
  PRIMARY KEY (standard_id, project_id)
);

CREATE TABLE tasks (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL,
  milestone_id TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  role TEXT NOT NULL DEFAULT 'developer',
  assignee TEXT NOT NULL DEFAULT '',
  priority INTEGER NOT NULL DEFAULT 2,  -- 1 highest
  status TEXT NOT NULL DEFAULT 'queued', -- queued | running | blocked | review | done | failed | paused
  deps TEXT NOT NULL DEFAULT '[]',
  reqs TEXT NOT NULL DEFAULT '[]',
  retries INTEGER NOT NULL DEFAULT 0,
  max_retries INTEGER NOT NULL DEFAULT 2,
  compactions INTEGER NOT NULL DEFAULT 0,
  handoffs INTEGER NOT NULL DEFAULT 0,
  branch TEXT NOT NULL DEFAULT '',
  pr_url TEXT NOT NULL DEFAULT '',
  context_note TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE INDEX tasks_project ON tasks(project_id, status);

CREATE TABLE sessions (
  id TEXT PRIMARY KEY,
  task_id TEXT NOT NULL,
  project_id TEXT NOT NULL,
  member_id TEXT NOT NULL,
  harness TEXT NOT NULL,
  model TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,          -- starting | running | paused | done | failed | cancelled | rate_limited
  started_at TEXT NOT NULL,
  ended_at TEXT NOT NULL DEFAULT '',
  used INTEGER NOT NULL DEFAULT 0,
  window INTEGER NOT NULL DEFAULT 0,
  measured INTEGER NOT NULL DEFAULT 0,
  compactions INTEGER NOT NULL DEFAULT 0,
  harness_session TEXT NOT NULL DEFAULT '',
  workdir TEXT NOT NULL DEFAULT '',
  resume_at TEXT NOT NULL DEFAULT '',
  exit_reason TEXT NOT NULL DEFAULT '',
  handoff TEXT NOT NULL DEFAULT ''
);
CREATE TABLE session_output (
  session_id TEXT NOT NULL,
  seq INTEGER NOT NULL,
  ts TEXT NOT NULL,
  kind TEXT NOT NULL,   -- text | tool | result | system | raw | error | input
  text TEXT NOT NULL,
  PRIMARY KEY (session_id, seq)
);

CREATE TABLE questions (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL,
  task_id TEXT NOT NULL DEFAULT '',
  session_id TEXT NOT NULL DEFAULT '',
  asker TEXT NOT NULL,
  title TEXT NOT NULL,
  why TEXT NOT NULL DEFAULT '',
  reqs TEXT NOT NULL DEFAULT '[]',
  blocking INTEGER NOT NULL DEFAULT 1,
  assumption TEXT NOT NULL DEFAULT '',
  recommendation TEXT NOT NULL DEFAULT '',
  options TEXT NOT NULL DEFAULT '[]',
  attachments TEXT NOT NULL DEFAULT '[]',
  status TEXT NOT NULL DEFAULT 'open',  -- open | answered | skipped
  answer TEXT NOT NULL DEFAULT '',
  choice TEXT NOT NULL DEFAULT '',
  answered_by TEXT NOT NULL DEFAULT '',
  answered_at TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);

CREATE TABLE reviews (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL,
  task_id TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending', -- pending | approved | changes | rejected
  assessment TEXT NOT NULL DEFAULT '{}',
  verdict TEXT NOT NULL DEFAULT '',
  comment TEXT NOT NULL DEFAULT '',
  reviewer TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  decided_at TEXT NOT NULL DEFAULT ''
);

CREATE TABLE rehearsals (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL,
  milestone_id TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL,
  kind TEXT NOT NULL DEFAULT 'tabletop', -- tabletop | uat
  status TEXT NOT NULL DEFAULT 'open',   -- open | approved | changes
  steps TEXT NOT NULL DEFAULT '[]',
  current INTEGER NOT NULL DEFAULT 0,
  host TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  decided_at TEXT NOT NULL DEFAULT ''
);
CREATE TABLE findings (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  rehearsal_id TEXT NOT NULL,
  step INTEGER NOT NULL,
  author TEXT NOT NULL,
  author_kind TEXT NOT NULL,
  text TEXT NOT NULL,
  change_request TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'open',
  created_at TEXT NOT NULL
);

CREATE TABLE option_sets (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL,
  title TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'open', -- open | decided
  generated_by TEXT NOT NULL DEFAULT '',
  value_lens TEXT NOT NULL DEFAULT '[]',
  options TEXT NOT NULL DEFAULT '[]',
  choice TEXT NOT NULL DEFAULT '[]',
  reason TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  decided_at TEXT NOT NULL DEFAULT ''
);

CREATE TABLE proposals (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  body TEXT NOT NULL,
  evidence TEXT NOT NULL DEFAULT '',
  author TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'open', -- open | accepted | rejected
  created_at TEXT NOT NULL
);

CREATE TABLE counters (name TEXT PRIMARY KEY, value INTEGER NOT NULL);
