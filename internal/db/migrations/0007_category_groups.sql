-- Category groups (see docs/groups.md): named, ordered labels like Housing or
-- Wealth building that categories sit under. Deleting a group ungroups its
-- categories; it never deletes them.
CREATE TABLE category_groups (
  id         INTEGER PRIMARY KEY,
  name       TEXT NOT NULL UNIQUE COLLATE NOCASE,
  sort_order INTEGER NOT NULL DEFAULT 0
);

-- NULL means ungrouped, which is every existing category.
ALTER TABLE categories
  ADD COLUMN group_id INTEGER REFERENCES category_groups (id) ON DELETE SET NULL;
