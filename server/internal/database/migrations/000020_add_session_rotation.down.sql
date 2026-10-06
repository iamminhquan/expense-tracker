DROP INDEX idx_sessions_previous_id;
ALTER TABLE sessions DROP COLUMN rotated_at;
ALTER TABLE sessions DROP COLUMN previous_id;
