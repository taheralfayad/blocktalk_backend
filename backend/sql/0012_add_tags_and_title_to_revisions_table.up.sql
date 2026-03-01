--- up

DROP TABLE tags_entry;

CREATE TABLE tags_entry_revision (
    entry_revision_id INTEGER,
    tag_id INTEGER,
    PRIMARY KEY (entry_revision_id, tag_id),
    FOREIGN KEY (entry_revision_id) REFERENCES entry_revision(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

ALTER TABLE entry DROP COLUMN title;

ALTER TABLE entry_revision ADD COLUMN title VARCHAR(255);