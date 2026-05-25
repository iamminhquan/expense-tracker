package repositories

import (
	"context"

	"github.com/iamminhquan/expense-tracker/internal/models"
	"gorm.io/gorm"
)

type BudgetRepository struct {
	db *gorm.DB
}

func NewBudgetRepository(db *gorm.DB) *BudgetRepository {
	return &BudgetRepository{db: db}
}

func (r *BudgetRepository) Create(ctx context.Context, budget *models.Budget) error {
	return r.db.WithContext(ctx).Create(budget).Error
}

func (r *BudgetRepository) ListByUser(ctx context.Context, userID uint) ([]models.Budget, error) {
	var budgets []models.Budget
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("year DESC, month DESC").
		Find(&budgets).Error; err != nil {
		return nil, err
	}
	return budgets, nil
}

func (r *BudgetRepository) FindByIDAndUser(ctx context.Context, id uint, userID uint) (*models.Budget, error) {
	var budget models.Budget
	if err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		First(&budget).Error; err != nil {
		return nil, err
	}
	return &budget, nil
}

// FindByPeriod looks up the budget for a specific category + period combination.
func (r *BudgetRepository) FindByPeriod(ctx context.Context, userID uint, categoryID uint, period string, year int, month *int) (*models.Budget, error) {
	var budget models.Budget
	q := r.db.WithContext(ctx).
		Where("user_id = ? AND category_id = ? AND period = ? AND year = ?", userID, categoryID, period, year)
	if month == nil {
		q = q.Where("month IS NULL")
	} else {
		q = q.Where("month = ?", *month)
	}
	if err := q.First(&budget).Error; err != nil {
		return nil, err
	}
	return &budget, nil
}

func (r *BudgetRepository) Save(ctx context.Context, budget *models.Budget) error {
	return r.db.WithContext(ctx).Save(budget).Error
}

func (r *BudgetRepository) Delete(ctx context.Context, budget *models.Budget) error {
	return r.db.WithContext(ctx).Delete(budget).Error
}
