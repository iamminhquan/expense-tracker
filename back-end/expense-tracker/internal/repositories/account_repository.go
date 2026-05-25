package repositories

import (
	"context"

	"github.com/iamminhquan/expense-tracker/internal/models"
	"gorm.io/gorm"
)

type AccountRepository struct {
	db *gorm.DB
}

func NewAccountRepository(db *gorm.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

func (r *AccountRepository) Create(ctx context.Context, account *models.Account) error {
	return r.db.WithContext(ctx).Create(account).Error
}

func (r *AccountRepository) ListByUser(ctx context.Context, userID uint) ([]models.Account, error) {
	var accounts []models.Account
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("is_default DESC, created_at ASC").
		Find(&accounts).Error; err != nil {
		return nil, err
	}
	return accounts, nil
}

func (r *AccountRepository) FindByIDAndUser(ctx context.Context, id uint, userID uint) (*models.Account, error) {
	var account models.Account
	if err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func (r *AccountRepository) FindDefaultByUser(ctx context.Context, userID uint) (*models.Account, error) {
	var account models.Account
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND is_default = true", userID).
		First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func (r *AccountRepository) Save(ctx context.Context, account *models.Account) error {
	return r.db.WithContext(ctx).Save(account).Error
}

func (r *AccountRepository) Delete(ctx context.Context, account *models.Account) error {
	return r.db.WithContext(ctx).Delete(account).Error
}
