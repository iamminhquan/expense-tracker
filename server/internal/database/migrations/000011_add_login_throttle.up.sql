-- Login throttling state, kept on users rather than in a table of its own:
-- only a real account is ever counted, so there is nothing to grow unbounded
-- and no rows to sweep up -- a lock simply expires where it sits.
--
-- locked_until is NULL for an account that is not locked; a past timestamp
-- reads the same way, which is what lets the lock lapse without a job.
ALTER TABLE users ADD COLUMN failed_login_attempts INT NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN locked_until TIMESTAMPTZ;
