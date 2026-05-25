package repositories

import (
	"context"

	"github.com/iamminhquan/expense-tracker/internal/models"
	"gorm.io/gorm"
)

type IncomeRepository struct {
	db *gorm.DB
}

func NewIncomeRepository(db *gorm.DB) *IncomeRepository {
	return &IncomeRepository{db: db}
}

func (r *IncomeRepository) Create(ctx context.Context, income *models.Income) error {
	return r.db.WithContext(ctx).Create(income).Error
}

func (r *IncomeRepository) ListByUser(ctx context.Context, userID uint) ([]models.Income, error) {
	var incomes []models.Income
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("date DESC").
		Order("created_at DESC").
		Find(&incomes).Error; err != nil {
		return nil, err
	}
	return incomes, nil
}

func (r *IncomeRepository) FindByIDAndUser(ctx context.Context, id uint, userID uint) (*models.Income, error) {
	var income models.Income
	if err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		First(&income).Error; err != nil {
		return nil, err
	}
	return &income, nil
}

func (r *IncomeRepository) Save(ctx context.Context, income *models.Income) error {
	return r.db.WithContext(ctx).Save(income).Error
}

func (r *IncomeRepository) Delete(ctx context.Context, income *models.Income) error {
	return r.db.WithContext(ctx).Delete(income).Error
}
