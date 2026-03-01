--- down

DROP TABLE entry_interactions;
DROP TABLE conversation_interactions;

--- why the hell did i add this in the first place lol
--- stupid stupid stupid
ALTER TABLE entry ADD COLUMN upvotes INTEGER NOT NULL DEFAULT 0;
ALTER TABLE entry ADD COLUMN downvotes INTEGER NOT NULL DEFAULT 0;
ALTER TABLE conversation ADD COLUMN upvotes INTEGER NOT NULL DEFAULT 0;
ALTER TABLE conversation ADD COLUMN downvotes INTEGER NOT NULL DEFAULT 0;