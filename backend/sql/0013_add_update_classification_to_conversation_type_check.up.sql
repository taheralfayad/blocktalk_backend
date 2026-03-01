ALTER TABLE conversation
DROP CONSTRAINT conversation_type_check;

ALTER TABLE conversation
ADD CONSTRAINT conversation_type_check
CHECK (type IN ('opinion', 'source', 'update'));