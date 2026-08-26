-- created_at lets the session list be sorted and lets a user tell "signed in
-- just now" from "signed in last week" -- DEFAULT now() backfills every
-- existing row instead of leaving it NULL.
ALTER TABLE sessions ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT now();

-- user_agent is nullable: a session created before this migration has none,
-- and there is no way to backfill it.
ALTER TABLE sessions ADD COLUMN user_agent TEXT;
