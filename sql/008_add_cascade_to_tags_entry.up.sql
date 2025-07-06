--- up

ALTER TABLE tags_entry DROP CONSTRAINT tags_entry_entry_id_fkey;
ALTER TABLE tags_entry DROP CONSTRAINT tags_entry_tag_id_fkey;

ALTER TABLE tags_entry
  ADD CONSTRAINT tags_entry_entry_id_fkey
    FOREIGN KEY (entry_id) REFERENCES entry(id) ON DELETE CASCADE;

ALTER TABLE tags_entry
  ADD CONSTRAINT tags_entry_tag_id_fkey
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE;