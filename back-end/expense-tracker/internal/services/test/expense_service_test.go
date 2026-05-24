package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/iamminhquan/expense-tracker/internal/models"
	"github.com/iamminhquan/expense-tracker/internal/services"
	"gorm.io/gorm"
)

type fakeExpenseRepository struct {
	nextID   uint
	expenses map[uint]*models.Expense
}

func newFakeExpenseRepository() *fakeExpenseRepository {
	return &fakeExpenseRepository{
		nextID:   1,
		expenses: make(map[uint]*models.Expense),
	}
}

func (r *fakeExpenseRepository) Create(_ context.Context, expense *models.Expense) error {
	expense.ID = r.nextID
	r.nextID++
	copiedExpense := *expense
	r.expenses[expense.ID] = &copiedExpense
	return nil
}

func (r *fakeExpenseRepository) ListByUser(_ context.Context, userID uint) ([]models.Expense, error) {
	expenses := make([]models.Expense, 0)
	for _, expense := range r.expenses {
		if expense.UserID == userID {
			expenses = append(expenses, *expense)
		}
	}
	return expenses, nil
}

func (r *fakeExpenseRepository) FindByIDAndUser(_ context.Context, id uint, userID uint) (*models.Expense, error) {
	expense, exists := r.expenses[id]
	if !exists || expense.UserID != userID {
		return nil, gorm.ErrRecordNotFound
	}
	return expense, nil
}

func (r *fakeExpenseRepository) Save(_ context.Context, expense *models.Expense) error {
	copiedExpense := *expense
	r.expenses[expense.ID] = &copiedExpense
	return nil
}

func (r *fakeExpenseRepository) Delete(_ context.Context, expense *models.Expense) error {
	delete(r.expenses, expense.ID)
	return nil
}

func TestExpenseCreateNormalizesInput(t *testing.T) {
	service := services.NewExpenseService(newFakeExpenseRepository())
	date := time.Date(2026, 5, 24, 0, 0, 0, 0, time.UTC)

	expense, err := service.Create(context.Background(), 7, services.ExpenseInput{
		Amount:        120000,
		Description:   " Lunch ",
		Category:      " Food ",
		Date:          date,
		PaymentMethod: " Cash ",
		Note:          " Team meal ",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if expense.UserID != 7 {
		t.Fatalf("Create() user id = %d, want 7", expense.UserID)
	}
	if expense.Description != "Lunch" || expense.Category != "Food" || expense.PaymentMethod != "Cash" || expense.Note != "Team meal" {
		t.Fatalf("Create() did not trim input: %#v", expense)
	}
}

func TestExpenseRejectsInvalidInput(t *testing.T) {
	service := services.NewExpenseService(newFakeExpenseRepository())
	date := time.Date(2026, 5, 24, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name  string
		input services.ExpenseInput
	}{
		{name: "missing amount", input: services.ExpenseInput{Description: "Lunch", Category: "Food", Date: date}},
		{name: "missing description", input: services.ExpenseInput{Amount: 120000, Category: "Food", Date: date}},
		{name: "missing category", input: services.ExpenseInput{Amount: 120000, Description: "Lunch", Date: date}},
		{name: "missing date", input: services.ExpenseInput{Amount: 120000, Description: "Lunch", Category: "Food"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Create(context.Background(), 7, tt.input)
			if !errors.Is(err, services.ErrInvalidExpenseInput) {
				t.Fatalf("Create() error = %v, want %v", err, services.ErrInvalidExpenseInput)
			}
		})
	}
}

func TestExpenseOperationsAreScopedToUser(t *testing.T) {
	service := services.NewExpenseService(newFakeExpenseRepository())
	date := time.Date(2026, 5, 24, 0, 0, 0, 0, time.UTC)

	expense, err := service.Create(context.Background(), 7, services.ExpenseInput{
		Amount:      120000,
		Description: "Lunch",
		Category:    "Food",
		Date:        date,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if _, err := service.Update(context.Background(), 8, expense.ID, services.ExpenseInput{
		Amount:      100000,
		Description: "Dinner",
		Category:    "Food",
		Date:        date,
	}); !errors.Is(err, services.ErrExpenseNotFound) {
		t.Fatalf("Update() error = %v, want %v", err, services.ErrExpenseNotFound)
	}

	if err := service.Delete(context.Background(), 8, expense.ID); !errors.Is(err, services.ErrExpenseNotFound) {
		t.Fatalf("Delete() error = %v, want %v", err, services.ErrExpenseNotFound)
	}

	expenses, err := service.List(context.Background(), 7)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(expenses) != 1 {
		t.Fatalf("List() length = %d, want 1", len(expenses))
	}
}
