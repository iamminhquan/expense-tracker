-- Per-user light/dark preference. 'auto' means "follow the operating
-- system", and is resolved purely in CSS (a prefers-color-scheme media
-- query) rather than server-side, so the server never has to guess what the
-- user's OS is set to.
ALTER TABLE users ADD COLUMN theme TEXT NOT NULL DEFAULT 'auto';

ALTER TABLE users ADD CONSTRAINT users_theme_check
    CHECK (theme IN ('auto', 'light', 'dark'));
