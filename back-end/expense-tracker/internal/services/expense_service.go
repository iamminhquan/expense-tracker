package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/iamminhquan/expense-tracker/internal/models"
	"gorm.io/gorm"
)

var (
	ErrInvalidExpenseInput = errors.New("amount, description, category, and date are required")
	ErrExpenseNotFound     = errors.New("expense not found")
)

type expenseRepository interface {
	Create(ctx context.Context, expense *models.Expense) error
	ListByUser(ctx context.Context, userID uint) ([]models.Expense, error)
	FindByIDAndUser(ctx context.Context, id uint, userID uint) (*models.Expense, error)
	Save(ctx context.Context, expense *models.Expense) error
	Delete(ctx context.Context, expense *models.Expense) error
}

type ExpenseService struct {
	repo expenseRepository
}

type ExpenseInput struct {
	Amount          float64
	Currency        string
	Description     string
	Category        string
	Date            time.Time
	PaymentMethod   string
	IsRecurring     bool
	RecurringPeriod string
	Note            string
}

func NewExpenseService(repo expenseRepository) *ExpenseService {
	return &ExpenseService{repo: repo}
}

func (s *ExpenseService) Create(ctx context.Context, userID uint, input ExpenseInput) (*models.Expense, error) {
	normalized, err := normalizeExpenseInput(input)
	if err != nil {
		return nil, err
	}

	expense := &models.Expense{
		UserID:          userID,
		Amount:          normalized.Amount,
		Currency:        normalized.Currency,
		Description:     normalized.Description,
		Category:        normalized.Category,
		Date:            normalized.Date,
		PaymentMethod:   normalized.PaymentMethod,
		IsRecurring:     normalized.IsRecurring,
		RecurringPeriod: normalized.RecurringPeriod,
		Note:            normalized.Note,
	}

	if err := s.repo.Create(ctx, expense); err != nil {
		return nil, err
	}

	return expense, nil
}

func (s *ExpenseService) List(ctx context.Context, userID uint) ([]models.Expense, error) {
	return s.repo.ListByUser(ctx, userID)
}

func (s *ExpenseService) Update(ctx context.Context, userID uint, id uint, input ExpenseInput) (*models.Expense, error) {
	normalized, err := normalizeExpenseInput(input)
	if err != nil {
		return nil, err
	}

	expense, err := s.repo.FindByIDAndUser(ctx, id, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrExpenseNotFound
	}
	if err != nil {
		return nil, err
	}

	expense.Amount = normalized.Amount
	expense.Currency = normalized.Currency
	expense.Description = normalized.Description
	expense.Category = normalized.Category
	expense.Date = normalized.Date
	expense.PaymentMethod = normalized.PaymentMethod
	expense.IsRecurring = normalized.IsRecurring
	expense.RecurringPeriod = normalized.RecurringPeriod
	expense.Note = normalized.Note

	if err := s.repo.Save(ctx, expense); err != nil {
		return nil, err
	}

	return expense, nil
}

func (s *ExpenseService) Delete(ctx context.Context, userID uint, id uint) error {
	expense, err := s.repo.FindByIDAndUser(ctx, id, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrExpenseNotFound
	}
	if err != nil {
		return err
	}

	return s.repo.Delete(ctx, expense)
}

func normalizeExpenseInput(input ExpenseInput) (ExpenseInput, error) {
	currency := strings.TrimSpace(input.Currency)
	if currency == "" {
		currency = "VND"
	}

	normalized := ExpenseInput{
		Amount:          input.Amount,
		Currency:        currency,
		Description:     strings.TrimSpace(input.Description),
		Category:        strings.TrimSpace(input.Category),
		Date:            input.Date,
		PaymentMethod:   strings.TrimSpace(input.PaymentMethod),
		IsRecurring:     input.IsRecurring,
		RecurringPeriod: strings.TrimSpace(input.RecurringPeriod),
		Note:            strings.TrimSpace(input.Note),
	}

	if normalized.Amount <= 0 || normalized.Description == "" || normalized.Category == "" || normalized.Date.IsZero() {
		return ExpenseInput{}, ErrInvalidExpenseInput
	}

	return normalized, nil
}
