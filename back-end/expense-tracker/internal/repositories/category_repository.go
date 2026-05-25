package repositories

import (
	"context"

	"github.com/iamminhquan/expense-tracker/internal/models"
	"gorm.io/gorm"
)

type CategoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

// ListByUser returns system defaults (user_id IS NULL) + user's custom categories.
func (r *CategoryRepository) ListByUser(ctx context.Context, userID uint) ([]models.Category, error) {
	var categories []models.Category
	if err := r.db.WithContext(ctx).
		Where("user_id IS NULL OR user_id = ?", userID).
		Order("type ASC, name ASC").
		Find(&categories).Error; err != nil {
		return nil, err
	}
	return categories, nil
}

func (r *CategoryRepository) ListByUserAndType(ctx context.Context, userID uint, categoryType string) ([]models.Category, error) {
	var categories []models.Category
	if err := r.db.WithContext(ctx).
		Where("(user_id IS NULL OR user_id = ?) AND type = ?", userID, categoryType).
		Order("name ASC").
		Find(&categories).Error; err != nil {
		return nil, err
	}
	return categories, nil
}

func (r *CategoryRepository) FindByID(ctx context.Context, id uint) (*models.Category, error) {
	var category models.Category
	if err := r.db.WithContext(ctx).First(&category, id).Error; err != nil {
		return nil, err
	}
	return &category, nil
}

func (r *CategoryRepository) Create(ctx context.Context, category *models.Category) error {
	return r.db.WithContext(ctx).Create(category).Error
}

func (r *CategoryRepository) Save(ctx context.Context, category *models.Category) error {
	return r.db.WithContext(ctx).Save(category).Error
}

// Delete only allows deleting user-owned categories (user_id = userID).
func (r *CategoryRepository) Delete(ctx context.Context, category *models.Category) error {
	return r.db.WithContext(ctx).Delete(category).Error
}
