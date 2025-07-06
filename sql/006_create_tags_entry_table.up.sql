--- up 

CREATE TABLE tags_entry (
    entry_id INTEGER,
    tag_id INTEGER,
    PRIMARY KEY (entry_id, tag_id),
    FOREIGN KEY (entry_id) REFERENCES entry(id),
    FOREIGN KEY (tag_id) REFERENCES tags(id)
)