package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/iamminhquan/expense-tracker/internal/auth"
	"github.com/iamminhquan/expense-tracker/internal/models"
	"github.com/iamminhquan/expense-tracker/internal/services"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type fakeUserRepository struct {
	nextID uint
	users  map[string]*models.User
}

func newFakeUserRepository() *fakeUserRepository {
	return &fakeUserRepository{
		nextID: 1,
		users:  make(map[string]*models.User),
	}
}

func (r *fakeUserRepository) Create(_ context.Context, user *models.User) error {
	user.ID = r.nextID
	r.nextID++
	copiedUser := *user
	r.users[user.Email] = &copiedUser
	return nil
}

func (r *fakeUserRepository) EmailExists(_ context.Context, email string) (bool, error) {
	_, exists := r.users[email]
	return exists, nil
}

func (r *fakeUserRepository) FindByEmail(_ context.Context, email string) (*models.User, error) {
	user, exists := r.users[email]
	if !exists {
		return nil, gorm.ErrRecordNotFound
	}
	return user, nil
}

func (r *fakeUserRepository) FindByID(_ context.Context, id uint) (*models.User, error) {
	for _, user := range r.users {
		if user.ID == id {
			return user, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func TestRegisterCreatesUserAndToken(t *testing.T) {
	repo := newFakeUserRepository()
	service := services.NewUserService(repo, "test-secret")

	result, err := service.Register(context.Background(), services.RegisterInput{
		Name:     " Minh Quan ",
		Email:    "Quan@Example.COM ",
		Password: "strong-password",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if result.User.ID == 0 {
		t.Fatal("Register() returned user without id")
	}
	if result.User.Name != "Minh Quan" {
		t.Fatalf("Register() name = %q", result.User.Name)
	}
	if result.User.Email != "quan@example.com" {
		t.Fatalf("Register() email = %q", result.User.Email)
	}
	if result.User.Password == "strong-password" {
		t.Fatal("Register() stored plaintext password")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(result.User.Password), []byte("strong-password")); err != nil {
		t.Fatalf("Register() password hash mismatch: %v", err)
	}

	userID, err := auth.ParseToken(result.Token, "test-secret")
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}
	if userID != result.User.ID {
		t.Fatalf("token subject = %d, want %d", userID, result.User.ID)
	}
}

func TestRegisterRejectsInvalidInput(t *testing.T) {
	service := services.NewUserService(newFakeUserRepository(), "test-secret")

	tests := []struct {
		name  string
		input services.RegisterInput
	}{
		{name: "missing name", input: services.RegisterInput{Email: "a@example.com", Password: "strong-password"}},
		{name: "invalid email", input: services.RegisterInput{Name: "Quan", Email: "invalid", Password: "strong-password"}},
		{name: "short password", input: services.RegisterInput{Name: "Quan", Email: "a@example.com", Password: "short"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Register(context.Background(), tt.input)
			if !errors.Is(err, services.ErrInvalidUserInput) {
				t.Fatalf("Register() error = %v, want %v", err, services.ErrInvalidUserInput)
			}
		})
	}
}

func TestRegisterRejectsDuplicateEmail(t *testing.T) {
	service := services.NewUserService(newFakeUserRepository(), "test-secret")
	input := services.RegisterInput{Name: "Quan", Email: "quan@example.com", Password: "strong-password"}

	if _, err := service.Register(context.Background(), input); err != nil {
		t.Fatalf("first Register() error = %v", err)
	}
	if _, err := service.Register(context.Background(), input); !errors.Is(err, services.ErrEmailAlreadyExists) {
		t.Fatalf("second Register() error = %v, want %v", err, services.ErrEmailAlreadyExists)
	}
}

func TestLoginReturnsTokenForValidCredentials(t *testing.T) {
	service := services.NewUserService(newFakeUserRepository(), "test-secret")
	registered, err := service.Register(context.Background(), services.RegisterInput{
		Name:     "Quan",
		Email:    "quan@example.com",
		Password: "strong-password",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	result, err := service.Login(context.Background(), services.LoginInput{
		Email:    "QUAN@example.com",
		Password: "strong-password",
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if result.User.ID != registered.User.ID {
		t.Fatalf("Login() user id = %d, want %d", result.User.ID, registered.User.ID)
	}
	if _, err := auth.ParseToken(result.Token, "test-secret"); err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}
}

func TestLoginRejectsInvalidCredentials(t *testing.T) {
	service := services.NewUserService(newFakeUserRepository(), "test-secret")

	_, err := service.Login(context.Background(), services.LoginInput{
		Email:    "missing@example.com",
		Password: "strong-password",
	})
	if !errors.Is(err, services.ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, want %v", err, services.ErrInvalidCredentials)
	}
}
