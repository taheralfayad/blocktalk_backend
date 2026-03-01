--- up

CREATE TABLE entry_revision (
    id SERIAL PRIMARY KEY,
    entry_id INTEGER NOT NULL,
    creator_id INTEGER NOT NULL,
    content TEXT NOT NULL,
    revision_number INTEGER NOT NULL,
    date_created TIMESTAMP DEFAULT now(),
    FOREIGN KEY (entry_id) REFERENCES entry(id) ON DELETE CASCADE,
    FOREIGN KEY (creator_id) REFERENCES users(id) ON DELETE CASCADE
);

ALTER TABLE entry DROP COLUMN content;