--- up

ALTER TABLE tags ADD COLUMN classification VARCHAR(50);
UPDATE tags SET classification = 'Zoning';
ALTER TABLE tags ALTER COLUMN classification SET NOT NULL;
