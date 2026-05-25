package database

import (
	"github.com/iamminhquan/expense-tracker/internal/config"
	"github.com/iamminhquan/expense-tracker/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect(cfg config.Config) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(
		&models.User{},
		&models.Account{},
		&models.Category{},
		&models.Expense{},
		&models.Income{},
		&models.Budget{},
		&models.SavingsGoal{},
	); err != nil {
		return nil, err
	}

	if err := seedDefaultCategories(db); err != nil {
		return nil, err
	}

	return db, nil
}

func seedDefaultCategories(db *gorm.DB) error {
	defaults := []models.Category{
		// Expense categories
		{Name: "Ăn uống", Type: models.CategoryTypeExpense, Icon: "utensils", Color: "#f97316"},
		{Name: "Đi lại", Type: models.CategoryTypeExpense, Icon: "car", Color: "#3b82f6"},
		{Name: "Giải trí", Type: models.CategoryTypeExpense, Icon: "gamepad-2", Color: "#a855f7"},
		{Name: "Mua sắm", Type: models.CategoryTypeExpense, Icon: "shopping-bag", Color: "#ec4899"},
		{Name: "Sức khỏe", Type: models.CategoryTypeExpense, Icon: "heart-pulse", Color: "#ef4444"},
		{Name: "Giáo dục", Type: models.CategoryTypeExpense, Icon: "book-open", Color: "#eab308"},
		{Name: "Hóa đơn & Tiện ích", Type: models.CategoryTypeExpense, Icon: "zap", Color: "#06b6d4"},
		{Name: "Nhà ở", Type: models.CategoryTypeExpense, Icon: "home", Color: "#22c55e"},
		{Name: "Du lịch", Type: models.CategoryTypeExpense, Icon: "plane", Color: "#0ea5e9"},
		{Name: "Khác", Type: models.CategoryTypeExpense, Icon: "ellipsis", Color: "#6b7280"},

		// Income categories
		{Name: "Lương", Type: models.CategoryTypeIncome, Icon: "briefcase", Color: "#22c55e"},
		{Name: "Freelance", Type: models.CategoryTypeIncome, Icon: "laptop", Color: "#3b82f6"},
		{Name: "Đầu tư", Type: models.CategoryTypeIncome, Icon: "trending-up", Color: "#10b981"},
		{Name: "Kinh doanh", Type: models.CategoryTypeIncome, Icon: "store", Color: "#f97316"},
		{Name: "Quà tặng", Type: models.CategoryTypeIncome, Icon: "gift", Color: "#ec4899"},
		{Name: "Khác", Type: models.CategoryTypeIncome, Icon: "ellipsis", Color: "#6b7280"},
	}

	for i := range defaults {
		cat := defaults[i]
		// Only insert if not already present (idempotent seed)
		db.Where(models.Category{Name: cat.Name, Type: cat.Type, UserID: nil}).
			FirstOrCreate(&cat)
	}

	return nil
}
