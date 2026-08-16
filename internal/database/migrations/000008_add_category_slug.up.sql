-- Give the shared default categories a stable, language-independent key.
--
-- Until now the only handle on a default category was its Vietnamese name,
-- which the UI also displayed. That made the display string load-bearing:
-- GetDefaultCategoryForReassignment finds the category that adopts orphaned
-- transactions by matching name = 'Khác', so translating the UI would have
-- broken category deletion. slug separates the two roles.
--
-- name moves to English at the same time rather than staying Vietnamese,
-- because two things still read it: ListCategoriesForUser orders by it (a
-- Vietnamese name behind an English label would sort the list in an order
-- the screen doesn't explain), and idx_categories_user_type_name uses it to
-- stop a user creating a category that collides with a default one.

ALTER TABLE categories ADD COLUMN slug TEXT;

UPDATE categories SET slug = 'food_drink',    name = 'Food & Drink'  WHERE user_id IS NULL AND type = 'expense' AND name = 'Ăn uống';
UPDATE categories SET slug = 'transport',     name = 'Transport'     WHERE user_id IS NULL AND type = 'expense' AND name = 'Đi lại';
UPDATE categories SET slug = 'entertainment', name = 'Entertainment' WHERE user_id IS NULL AND type = 'expense' AND name = 'Giải trí';
UPDATE categories SET slug = 'bills',         name = 'Bills'         WHERE user_id IS NULL AND type = 'expense' AND name = 'Hóa đơn';
UPDATE categories SET slug = 'health',        name = 'Health'        WHERE user_id IS NULL AND type = 'expense' AND name = 'Sức khỏe';
UPDATE categories SET slug = 'shopping',      name = 'Shopping'      WHERE user_id IS NULL AND type = 'expense' AND name = 'Mua sắm';
UPDATE categories SET slug = 'other',         name = 'Other'         WHERE user_id IS NULL AND type = 'expense' AND name = 'Khác';
UPDATE categories SET slug = 'salary',        name = 'Salary'        WHERE user_id IS NULL AND type = 'income'  AND name = 'Lương';
UPDATE categories SET slug = 'bonus',         name = 'Bonus'         WHERE user_id IS NULL AND type = 'income'  AND name = 'Thưởng';

-- "Thu nhập khác" is the easy one to miss. Migration 000006 only deletes it
-- when no transaction references it (deleting a referenced row would violate
-- the transactions FK), so a fresh database no longer has it but any account
-- that used it still does. Without this it would be the one default row left
-- showing a Vietnamese name. A no-op where the row is already gone.
UPDATE categories SET slug = 'other_income',  name = 'Other income'  WHERE user_id IS NULL AND type = 'income'  AND name = 'Thu nhập khác';

-- Partial index: user-created categories keep slug NULL and are free to
-- collide with each other, but no two rows may claim the same default.
CREATE UNIQUE INDEX idx_categories_slug ON categories (slug) WHERE slug IS NOT NULL;
