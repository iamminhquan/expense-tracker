-- sessions.id is the refresh token itself (the httpOnly cookie's value), so it
-- must never leave the server: the settings page listed it, which put every
-- active refresh token -- the current one included -- in reach of page
-- scripts and made the cookie's httpOnly flag moot. public_id is the
-- identifier a client may see and send back to revoke a session.
--
-- The volatile DEFAULT is evaluated per row, so every existing session gets its
-- own value as the column is added.
ALTER TABLE sessions ADD COLUMN public_id UUID NOT NULL DEFAULT gen_random_uuid();
CREATE UNIQUE INDEX idx_sessions_public_id ON sessions(public_id);
