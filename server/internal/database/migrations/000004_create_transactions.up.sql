CREATE TABLE transactions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category_id BIGINT NOT NULL REFERENCES categories(id),
    amount BIGINT NOT NULL CHECK (amount > 0),
    type TEXT NOT NULL CHECK (type IN ('expense', 'income')),
    description TEXT NOT NULL DEFAULT '',
    occurred_on DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_transactions_user_id_occurred_on ON transactions(user_id, occurred_on);
