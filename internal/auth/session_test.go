package auth_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"expensetracker/internal/auth"
	"expensetracker/internal/database"
	"expensetracker/internal/sqlcgen"
)

func setupTestUser(t *testing.T, q *sqlcgen.Queries) int64 {
	t.Helper()
	ctx := context.Background()
	email := "session-test@example.com"
	pool := testPool(t)
	_, _ = pool.Exec(ctx, "DELETE FROM users WHERE email = $1", email)
	user, err := q.CreateUser(ctx, sqlcgen.CreateUserParams{
		Email:        email,
		PasswordHash: "hashed",
		Name:         "Session Test",
		Username:     "session_test",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return user.ID
}

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := database.NewPool(context.Background(), dsn)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestSessionLifecycle(t *testing.T) {
	pool := testPool(t)
	q := sqlcgen.New(pool)
	userID := setupTestUser(t, q)
	mgr := auth.NewManager(q)
	ctx := context.Background()

	token, expiresAt, err := mgr.CreateSession(ctx, userID)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	if !expiresAt.After(time.Now()) {
		t.Fatal("expected expiry to be in the future")
	}

	gotUserID, err := mgr.ValidateSession(ctx, token)
	if err != nil {
		t.Fatalf("validate session: %v", err)
	}
	if gotUserID != userID {
		t.Fatalf("expected user id %d, got %d", userID, gotUserID)
	}

	if err := mgr.DeleteSession(ctx, token); err != nil {
		t.Fatalf("delete session: %v", err)
	}

	if _, err := mgr.ValidateSession(ctx, token); err == nil {
		t.Fatal("expected validate to fail after delete")
	}
}
