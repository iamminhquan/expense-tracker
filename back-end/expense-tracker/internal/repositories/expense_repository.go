package repositories

import (
	"context"

	"github.com/iamminhquan/expense-tracker/internal/models"
	"gorm.io/gorm"
)

type ExpenseRepository struct {
	db *gorm.DB
}

func NewExpenseRepository(db *gorm.DB) *ExpenseRepository {
	return &ExpenseRepository{db: db}
}

func (r *ExpenseRepository) Create(ctx context.Context, expense *models.Expense) error {
	return r.db.WithContext(ctx).Create(expense).Error
}

func (r *ExpenseRepository) ListByUser(ctx context.Context, userID uint) ([]models.Expense, error) {
	var expenses []models.Expense
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("date DESC").
		Order("created_at DESC").
		Find(&expenses).Error; err != nil {
		return nil, err
	}

	return expenses, nil
}

func (r *ExpenseRepository) FindByIDAndUser(ctx context.Context, id uint, userID uint) (*models.Expense, error) {
	var expense models.Expense
	if err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		First(&expense).Error; err != nil {
		return nil, err
	}

	return &expense, nil
}

func (r *ExpenseRepository) Save(ctx context.Context, expense *models.Expense) error {
	return r.db.WithContext(ctx).Save(expense).Error
}

func (r *ExpenseRepository) Delete(ctx context.Context, expense *models.Expense) error {
	return r.db.WithContext(ctx).Delete(expense).Error
}
