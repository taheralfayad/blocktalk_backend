CREATE TABLE conversation(
    id SERIAL PRIMARY KEY,
    user_id INTEGER,
    entry_id INTEGER,
    parent_id INTEGER,
    context Text,
    upvotes INTEGER NOT NULL DEFAULT 0,
    downvotes INTEGER NOT NULL DEFAULT 0,
    type VARCHAR(20) NOT NULL CHECK (type IN ('opinion', 'source')),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (entry_id) REFERENCES entry(id) ON DELETE CASCADE,
    FOREIGN KEY (parent_id) REFERENCES conversation(id) ON DELETE CASCADE
);