CREATE TABLE entry_contributors (
    entry_id INTEGER,
    user_id INTEGER,
    PRIMARY KEY (user_id, entry_id),
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (entry_id) REFERENCES entry(id)
);