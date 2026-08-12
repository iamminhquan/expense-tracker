-- Replace the placeholder 8-category seed from 000005 with the exact 9
-- categories and hex values SPEC.md section 4.3 specifies, and add the
-- constraints the redesigned category UI needs: a fixed 9-color palette (8
-- user-selectable swatches + the reserved gray for "Khác"), and a per-user
-- uniqueness rule so a user can't have two categories of the same type
-- sharing a name (Postgres treats NULL user_id rows as always-distinct in
-- a unique index, so this only constrains real per-user rows, not the
-- shared defaults).
DELETE FROM categories WHERE user_id IS NULL;

INSERT INTO categories (user_id, name, type, color) VALUES
    (NULL, 'Ăn uống', 'expense', '#D97757'),
    (NULL, 'Đi lại', 'expense', '#5B8DEF'),
    (NULL, 'Mua sắm', 'expense', '#8B7BD8'),
    (NULL, 'Hóa đơn', 'expense', '#6BA292'),
    (NULL, 'Giải trí', 'expense', '#E0A82E'),
    (NULL, 'Sức khỏe', 'expense', '#D97AA0'),
    (NULL, 'Khác', 'expense', '#A1A1AA'),
    (NULL, 'Lương', 'income', '#4FA871'),
    (NULL, 'Thưởng', 'income', '#7CA65C');

ALTER TABLE categories ADD CONSTRAINT categories_color_check CHECK (
    color IN ('#D97757', '#5B8DEF', '#8B7BD8', '#6BA292', '#E0A82E', '#D97AA0', '#4FA871', '#7CA65C', '#A1A1AA')
);

CREATE UNIQUE INDEX idx_categories_user_type_name ON categories (user_id, type, name);
