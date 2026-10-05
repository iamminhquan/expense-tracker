-- Fully reversible, unlike 000006: every value this migration overwrote is
-- recoverable from the slug it wrote alongside, so the names go back exactly
-- as they were rather than best-effort.
DROP INDEX idx_categories_slug;

UPDATE categories SET name = 'Ăn uống'       WHERE slug = 'food_drink';
UPDATE categories SET name = 'Đi lại'        WHERE slug = 'transport';
UPDATE categories SET name = 'Giải trí'      WHERE slug = 'entertainment';
UPDATE categories SET name = 'Hóa đơn'       WHERE slug = 'bills';
UPDATE categories SET name = 'Sức khỏe'      WHERE slug = 'health';
UPDATE categories SET name = 'Mua sắm'       WHERE slug = 'shopping';
UPDATE categories SET name = 'Khác'          WHERE slug = 'other';
UPDATE categories SET name = 'Lương'         WHERE slug = 'salary';
UPDATE categories SET name = 'Thưởng'        WHERE slug = 'bonus';
UPDATE categories SET name = 'Thu nhập khác' WHERE slug = 'other_income';

ALTER TABLE categories DROP COLUMN slug;
