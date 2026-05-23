package services

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/iamminhquan/expense-tracker/internal/auth"
	"github.com/iamminhquan/expense-tracker/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidUserInput   = errors.New("name, valid email, and password of at least 8 characters are required")
	ErrEmailAlreadyExists = errors.New("email already exists")
)

type userRepository interface {
	Create(ctx context.Context, user *models.User) error
	EmailExists(ctx context.Context, email string) (bool, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindByID(ctx context.Context, id uint) (*models.User, error)
}

type UserService struct {
	repo       userRepository
	authSecret string
}

type RegisterInput struct {
	Name     string
	Email    string
	Password string
}

type LoginInput struct {
	Email    string
	Password string
}

type AuthResult struct {
	User  *models.User
	Token string
}

func NewUserService(repo userRepository, authSecret string) *UserService {
	return &UserService{repo: repo, authSecret: authSecret}
}

func (s *UserService) Register(ctx context.Context, input RegisterInput) (*AuthResult, error) {
	name := strings.TrimSpace(input.Name)
	email := normalizeEmail(input.Email)
	password := input.Password

	if name == "" || !isValidEmail(email) || len(password) < 8 {
		return nil, ErrInvalidUserInput
	}

	exists, err := s.repo.EmailExists(ctx, email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrEmailAlreadyExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Name:     name,
		Email:    email,
		Password: string(hashedPassword),
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	token, err := auth.GenerateToken(user.ID, s.authSecret, 24*time.Hour)
	if err != nil {
		return nil, err
	}

	return &AuthResult{User: user, Token: token}, nil
}

func (s *UserService) Login(ctx context.Context, input LoginInput) (*AuthResult, error) {
	email := normalizeEmail(input.Email)
	password := input.Password
	if email == "" || password == "" {
		return nil, ErrInvalidCredentials
	}

	user, err := s.repo.FindByEmail(ctx, email)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, err := auth.GenerateToken(user.ID, s.authSecret, 24*time.Hour)
	if err != nil {
		return nil, err
	}

	return &AuthResult{User: user, Token: token}, nil
}

func (s *UserService) FindByID(ctx context.Context, id uint) (*models.User, error) {
	return s.repo.FindByID(ctx, id)
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func isValidEmail(email string) bool {
	parsed, err := mail.ParseAddress(email)
	return err == nil && parsed.Address == email
}
