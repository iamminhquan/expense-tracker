DROP INDEX idx_categories_user_type_name;
ALTER TABLE categories DROP CONSTRAINT categories_color_check;

DELETE FROM categories WHERE user_id IS NULL;

INSERT INTO categories (user_id, name, type, color) VALUES
    (NULL, 'Ăn uống', 'expense', '#ef4444'),
    (NULL, 'Di chuyển', 'expense', '#f97316'),
    (NULL, 'Giải trí', 'expense', '#eab308'),
    (NULL, 'Hóa đơn', 'expense', '#8b5cf6'),
    (NULL, 'Sức khỏe', 'expense', '#ec4899'),
    (NULL, 'Mua sắm', 'expense', '#06b6d4'),
    (NULL, 'Lương', 'income', '#22c55e'),
    (NULL, 'Thu nhập khác', 'income', '#10b981');
