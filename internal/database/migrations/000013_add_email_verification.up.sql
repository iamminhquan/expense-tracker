-- email_verified gates nothing by itself -- see the layout banner in
-- internal/handlers -- it only tracks whether the address on file has ever
-- been proven reachable. pending_email holds a requested new address until
-- its own link is clicked, so a mistyped change can never overwrite the one
-- address a locked-out owner can still be reached at.
ALTER TABLE users ADD COLUMN email_verified BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE users ADD COLUMN pending_email TEXT;

-- Every account that already exists is grandfathered in as verified: this
-- column did not exist when they registered, and there is no reason to make
-- someone who has used the app for months suddenly see a "verify your
-- email" banner for a check that postdates their signup.
UPDATE users SET email_verified = true;

-- One token table serves both entry points -- a fresh signup and a settings
-- email change -- because both ultimately ask the same question: prove you
-- can read mail at this address. email is the address being proven, which
-- for a signup equals users.email already and for a change equals
-- pending_email; confirming a token copies it onto users.email either way.
CREATE TABLE email_verification_tokens (
    token TEXT PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_email_verification_tokens_user_id ON email_verification_tokens(user_id);
