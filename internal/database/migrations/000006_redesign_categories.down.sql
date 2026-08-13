-- This is a best-effort reverse of the data-preserving 000006 up-migration.
-- It undoes the schema constraints and the known default-category renames/
-- recolors. It intentionally does NOT attempt to undo the custom-category
-- color reassignment or the duplicate-name-suffix dedup performed by the
-- up-migration's UPDATE statements -- those are lossy (the original values
-- aren't recorded anywhere), so a full reverse isn't possible without a
-- backup. This is honest and non-destructive, unlike a naive delete+
-- reinsert of the old defaults (which would violate the transactions FK
-- for any account with data referencing the new default rows).
DROP INDEX idx_categories_user_type_name;
ALTER TABLE categories DROP CONSTRAINT categories_color_check;

UPDATE categories SET color = '#ef4444' WHERE user_id IS NULL AND type = 'expense' AND name = 'Ăn uống';
UPDATE categories SET color = '#eab308' WHERE user_id IS NULL AND type = 'expense' AND name = 'Giải trí';
UPDATE categories SET color = '#8b5cf6' WHERE user_id IS NULL AND type = 'expense' AND name = 'Hóa đơn';
UPDATE categories SET color = '#ec4899' WHERE user_id IS NULL AND type = 'expense' AND name = 'Sức khỏe';
UPDATE categories SET color = '#06b6d4' WHERE user_id IS NULL AND type = 'expense' AND name = 'Mua sắm';
UPDATE categories SET color = '#22c55e' WHERE user_id IS NULL AND type = 'income' AND name = 'Lương';
UPDATE categories SET name = 'Di chuyển', color = '#f97316' WHERE user_id IS NULL AND type = 'expense' AND name = 'Đi lại';
UPDATE categories SET color = '#10b981' WHERE user_id IS NULL AND type = 'income' AND name = 'Thu nhập khác';

-- Remove the two brand-new defaults only if nothing references them yet
-- (safe no-op if any transaction or custom-category-reassignment already
-- used "Khác"/"Thưởng" -- those rows must stay for FK integrity).
DELETE FROM categories
WHERE user_id IS NULL AND type = 'expense' AND name = 'Khác'
  AND NOT EXISTS (SELECT 1 FROM transactions WHERE transactions.category_id = categories.id);

DELETE FROM categories
WHERE user_id IS NULL AND type = 'income' AND name = 'Thưởng'
  AND NOT EXISTS (SELECT 1 FROM transactions WHERE transactions.category_id = categories.id);
