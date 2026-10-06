-- A refresh now swaps the session's token for a new one (sessions.id is the
-- refresh token). previous_id keeps the token just replaced, and rotated_at
-- says when, so that:
--   * a request still carrying the old token a moment later (a second tab, a
--     retry after a dropped response) is answered rather than logged out, and
--   * the old token turning up after that moment is treated as a stolen copy
--     being replayed, and ends the session.
-- Only the latest replaced token is kept; an older one is just unknown.
ALTER TABLE sessions ADD COLUMN previous_id TEXT;
ALTER TABLE sessions ADD COLUMN rotated_at TIMESTAMPTZ;
CREATE UNIQUE INDEX idx_sessions_previous_id ON sessions(previous_id) WHERE previous_id IS NOT NULL;
