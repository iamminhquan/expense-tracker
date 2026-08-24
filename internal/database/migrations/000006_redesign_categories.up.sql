-- Move the placeholder 8-category seed from 000005 to the 9 shared default
-- categories and hex values the app ships with, and add the constraints the
-- redesigned category UI needs: a fixed 9-color palette (8
-- user-selectable swatches + the reserved gray for "Khác"), and a per-user
-- uniqueness rule so a user can't have two categories of the same type
-- sharing a name (Postgres treats NULL user_id rows as always-distinct in
-- a unique index, so this only constrains real per-user rows, not the
-- shared defaults).
--
-- Unlike an earlier version of this migration, this is data-preserving: it
-- UPDATEs existing default categories in place (keeping their ids, so any
-- transaction already pointing at one of them stays valid) instead of
-- deleting and re-inserting them, which would violate the transactions FK
-- (categories(id) has no ON DELETE clause) for any account with data.

-- Update the 6 existing default categories that keep the same name in the
-- new palette, in place (preserves their ids, so existing transactions
-- keep pointing at valid rows).
UPDATE categories SET color = '#D97757' WHERE user_id IS NULL AND type = 'expense' AND name = 'Ăn uống';
UPDATE categories SET color = '#8B7BD8' WHERE user_id IS NULL AND type = 'expense' AND name = 'Mua sắm';
UPDATE categories SET color = '#6BA292' WHERE user_id IS NULL AND type = 'expense' AND name = 'Hóa đơn';
UPDATE categories SET color = '#E0A82E' WHERE user_id IS NULL AND type = 'expense' AND name = 'Giải trí';
UPDATE categories SET color = '#D97AA0' WHERE user_id IS NULL AND type = 'expense' AND name = 'Sức khỏe';
UPDATE categories SET color = '#4FA871' WHERE user_id IS NULL AND type = 'income' AND name = 'Lương';

-- "Di chuyển" is renamed to "Đi lại" in the new palette (same semantic
-- category, transportation expenses) -- update in place rather than
-- delete+insert so existing transactions stay valid.
UPDATE categories SET name = 'Đi lại', color = '#5B8DEF' WHERE user_id IS NULL AND type = 'expense' AND name = 'Di chuyển';

-- "Thu nhập khác" has no equivalent in the new 9-category set (which pairs
-- "Lương" with "Thưởng" instead of a generic other-income category). On a
-- fresh install (or any account that never used it) it's safe to remove,
-- so a new database correctly ends up with exactly 9 defaults. On an
-- account that already has income transactions referencing it, deleting
-- it would violate the transactions FK, so it is kept (recolored to an
-- in-palette value distinct from "Thưởng") rather than deleted, and is no
-- longer offered by the application as one of the 9 defaults going
-- forward -- it just keeps working for accounts that already used it.
DELETE FROM categories
WHERE user_id IS NULL AND type = 'income' AND name = 'Thu nhập khác'
  AND NOT EXISTS (SELECT 1 FROM transactions WHERE transactions.category_id = categories.id);

UPDATE categories SET color = '#6BA292' WHERE user_id IS NULL AND type = 'income' AND name = 'Thu nhập khác';

-- Insert the two genuinely new defaults, but only if they don't already
-- exist (idempotency: a fresh database that never had the old 8-category
-- seed, or a database where this migration is somehow re-run, must not
-- get duplicate rows).
INSERT INTO categories (user_id, name, type, color)
SELECT NULL, 'Khác', 'expense', '#A1A1AA'
WHERE NOT EXISTS (
    SELECT 1 FROM categories WHERE user_id IS NULL AND type = 'expense' AND name = 'Khác'
);

INSERT INTO categories (user_id, name, type, color)
SELECT NULL, 'Thưởng', 'income', '#7CA65C'
WHERE NOT EXISTS (
    SELECT 1 FROM categories WHERE user_id IS NULL AND type = 'income' AND name = 'Thưởng'
);

-- Any pre-existing CUSTOM (user-owned) category whose color isn't one of
-- the new fixed palette's values gets deterministically reassigned to one
-- of the 8 user-selectable swatches (never the reserved gray, which is
-- reserved for the system "Khác") so the CHECK constraint below can be
-- added without deleting or corrupting anyone's categories. The mapping is
-- arbitrary but deterministic (based on row id), not a perceptual "nearest
-- color" match -- good enough since this only affects categories a user
-- created with the old free-form color picker before this migration.
UPDATE categories
SET color = (ARRAY['#D97757','#5B8DEF','#8B7BD8','#6BA292','#E0A82E','#D97AA0','#4FA871','#7CA65C'])[(id % 8) + 1]
WHERE user_id IS NOT NULL
  AND color NOT IN ('#D97757','#5B8DEF','#8B7BD8','#6BA292','#E0A82E','#D97AA0','#4FA871','#7CA65C','#A1A1AA');

-- De-duplicate any (user_id, type, name) collision among CUSTOM categories
-- before adding the uniqueness index below -- nothing previously prevented
-- a user from having two same-type categories sharing a name. Keeps the
-- oldest row's name as-is; later duplicates get a disambiguating numeric
-- suffix so no category or its transaction history is lost or merged.
WITH ranked AS (
    SELECT id, name,
           ROW_NUMBER() OVER (PARTITION BY user_id, type, name ORDER BY id) AS rn
    FROM categories
    WHERE user_id IS NOT NULL
)
UPDATE categories c
SET name = c.name || ' (' || ranked.rn || ')'
FROM ranked
WHERE c.id = ranked.id AND ranked.rn > 1;

ALTER TABLE categories ADD CONSTRAINT categories_color_check CHECK (
    color IN ('#D97757', '#5B8DEF', '#8B7BD8', '#6BA292', '#E0A82E', '#D97AA0', '#4FA871', '#7CA65C', '#A1A1AA')
);

CREATE UNIQUE INDEX idx_categories_user_type_name ON categories (user_id, type, name);
