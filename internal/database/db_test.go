package database_test

import (
	"context"
	"os"
	"testing"

	"expensetracker/internal/database"
	"expensetracker/internal/sqlcgen"
)

func TestCreateAndGetUser(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()

	pool, err := database.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	defer pool.Close()

	q := sqlcgen.New(pool)

	email := "test-user@example.com"
	_, _ = pool.Exec(ctx, "DELETE FROM users WHERE email = $1", email)

	created, err := q.CreateUser(ctx, sqlcgen.CreateUserParams{
		Email:        email,
		PasswordHash: "hashed",
		Name:         "Test User",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if created.Email != email {
		t.Fatalf("expected email %q, got %q", email, created.Email)
	}

	fetched, err := q.GetUserByEmail(ctx, email)
	if err != nil {
		t.Fatalf("get user by email: %v", err)
	}
	if fetched.ID != created.ID {
		t.Fatalf("expected id %d, got %d", created.ID, fetched.ID)
	}
}
