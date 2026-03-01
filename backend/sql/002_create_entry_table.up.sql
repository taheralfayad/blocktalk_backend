--- up

CREATE TABLE entry (
    id SERIAL PRIMARY KEY,
    address VARCHAR(60) NOT NULL,
    location GEOGRAPHY(Point, 4326) NOT NULL,
    content TEXT,
    upvotes INTEGER NOT NULL DEFAULT 0,
    downvotes INTEGER NOT NULL DEFAULT 0,
    views INTEGER NOT NULL DEFAULT 0,
    date_created TIMESTAMP DEFAULT now(),
    creator_id INTEGER,
    FOREIGN KEY (creator_id) REFERENCES users(id) ON DELETE CASCADE
);