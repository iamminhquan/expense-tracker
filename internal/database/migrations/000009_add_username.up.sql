-- Public, editable-later handle shown in the nav in place of the free-text
-- name collected at signup. Unlike name it must be unique and follow a
-- fixed shape, so it gets its own column and CHECK rather than reusing name.
--
-- Existing rows get a placeholder derived from id before the NOT NULL/UNIQUE
-- constraints land, purely so this migration doesn't fail against a database
-- that already has rows -- there is no real account data to preserve here.
ALTER TABLE users ADD COLUMN username TEXT;

UPDATE users SET username = 'user_' || id WHERE username IS NULL;

ALTER TABLE users ALTER COLUMN username SET NOT NULL;
ALTER TABLE users ADD CONSTRAINT users_username_key UNIQUE (username);
ALTER TABLE users ADD CONSTRAINT users_username_check
    CHECK (username ~ '^[a-z][a-z0-9_]{2,19}$');
