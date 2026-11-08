-- P9-08: reviews can be assigned to a human reviewer.
ALTER TABLE reviews ADD COLUMN assignee TEXT NOT NULL DEFAULT '';
