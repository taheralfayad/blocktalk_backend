--- up 
CREATE TABLE entry_interactions (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    entry_id INTEGER NOT NULL REFERENCES entry(id) ON DELETE CASCADE,
    interaction_type VARCHAR(50) NOT NULL CHECK (interaction_type IN ('upvote', 'downvote')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE (user_id, entry_id, interaction_type)
);

CREATE TABLE conversation_interactions (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    conversation_id INTEGER NOT NULL REFERENCES conversation(id) ON DELETE CASCADE,
    interaction_type VARCHAR(50) NOT NULL CHECK (interaction_type IN ('upvote', 'downvote')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE (user_id, conversation_id, interaction_type)
);

ALTER TABLE entry DROP COLUMN IF EXISTS upvotes;
ALTER TABLE entry DROP COLUMN IF EXISTS downvotes;
ALTER TABLE conversation DROP COLUMN IF EXISTS upvotes;
ALTER TABLE conversation DROP COLUMN IF EXISTS downvotes;