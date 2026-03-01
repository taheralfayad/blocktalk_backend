--- down

CREATE TABLE tags_entry (
    entry_id INTEGER,
    tag_id INTEGER,
    PRIMARY KEY (entry_id, tag_id),
    FOREIGN KEY (entry_id) REFERENCES entry(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
)

DROP TABLE tags_entry_revision;

ALTER TABLE entry ADD COLUMN title VARCHAR(255);

ALTER TABLE entry_revision DROP COLUMN title;