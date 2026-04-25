CREATE UNIQUE INDEX IF NOT EXISTS labels_name_lower_unique_idx ON labels (LOWER(name));
