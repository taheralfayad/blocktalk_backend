--- down

DROP TABLE IF EXISTS entry_revision;

ALTER TABLE entry ADD COLUMN content TEXT;