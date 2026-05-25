package repositories

import (
	"context"

	"github.com/iamminhquan/expense-tracker/internal/models"
	"gorm.io/gorm"
)

type SavingsGoalRepository struct {
	db *gorm.DB
}

func NewSavingsGoalRepository(db *gorm.DB) *SavingsGoalRepository {
	return &SavingsGoalRepository{db: db}
}

func (r *SavingsGoalRepository) Create(ctx context.Context, goal *models.SavingsGoal) error {
	return r.db.WithContext(ctx).Create(goal).Error
}

func (r *SavingsGoalRepository) ListByUser(ctx context.Context, userID uint) ([]models.SavingsGoal, error) {
	var goals []models.SavingsGoal
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at ASC").
		Find(&goals).Error; err != nil {
		return nil, err
	}
	return goals, nil
}

func (r *SavingsGoalRepository) FindByIDAndUser(ctx context.Context, id uint, userID uint) (*models.SavingsGoal, error) {
	var goal models.SavingsGoal
	if err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		First(&goal).Error; err != nil {
		return nil, err
	}
	return &goal, nil
}

func (r *SavingsGoalRepository) Save(ctx context.Context, goal *models.SavingsGoal) error {
	return r.db.WithContext(ctx).Save(goal).Error
}

func (r *SavingsGoalRepository) Delete(ctx context.Context, goal *models.SavingsGoal) error {
	return r.db.WithContext(ctx).Delete(goal).Error
}
