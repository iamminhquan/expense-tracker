package auth_test

import (
	"context"
	"errors"
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
	ctx := context.Background()

	token, expiresAt, err := auth.CreateSession(ctx, q, userID, "test-agent/1.0")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	if !expiresAt.After(time.Now()) {
		t.Fatal("expected expiry to be in the future")
	}

	session, err := q.GetSession(ctx, token)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if !session.UserAgent.Valid || session.UserAgent.String != "test-agent/1.0" {
		t.Fatalf("CreateSession(...) stored user agent = %+v, want %q", session.UserAgent, "test-agent/1.0")
	}

	refreshed, err := auth.RefreshSession(ctx, q, token)
	if err != nil {
		t.Fatalf("refresh session: %v", err)
	}
	if refreshed.UserID != userID {
		t.Fatalf("expected user id %d, got %d", userID, refreshed.UserID)
	}

	if err := auth.DeleteSession(ctx, q, refreshed.Token); err != nil {
		t.Fatalf("delete session: %v", err)
	}

	if _, err := auth.RefreshSession(ctx, q, refreshed.Token); !errors.Is(err, auth.ErrInvalidRefreshToken) {
		t.Fatalf("refresh after delete = %v, want ErrInvalidRefreshToken", err)
	}
}

func TestRefreshSessionRotatesTheToken(t *testing.T) {
	pool := testPool(t)
	q := sqlcgen.New(pool)
	userID := setupTestUser(t, q)
	ctx := context.Background()

	token, expiresAt, err := auth.CreateSession(ctx, q, userID, "agent")
	if err != nil {
		t.Fatal(err)
	}
	before, err := q.GetSession(ctx, token)
	if err != nil {
		t.Fatal(err)
	}

	first, err := auth.RefreshSession(ctx, q, token)
	if err != nil {
		t.Fatalf("first refresh: %v", err)
	}
	if first.Token == token {
		t.Fatal("refresh returned the same token, want a new one")
	}
	if !first.ExpiresAt.Equal(expiresAt) {
		t.Errorf("expiry moved from %v to %v; rotating must not extend a login", expiresAt, first.ExpiresAt)
	}

	after, err := q.GetSession(ctx, first.Token)
	if err != nil {
		t.Fatalf("the new token names no session: %v", err)
	}
	if after.PublicID != before.PublicID || after.CreatedAt != before.CreatedAt {
		t.Error("rotating changed the session's public id or creation time; it must stay the same session")
	}

	// The new token rotates again.
	second, err := auth.RefreshSession(ctx, q, first.Token)
	if err != nil {
		t.Fatalf("second refresh: %v", err)
	}
	if second.Token == first.Token {
		t.Fatal("second refresh returned the same token")
	}
}

// A request carrying the token just replaced is a racing tab or a retry, not
// an attack: it is answered with the current token, and rotates nothing.
func TestRefreshSessionAnswersAReplayInsideTheGracePeriod(t *testing.T) {
	pool := testPool(t)
	q := sqlcgen.New(pool)
	userID := setupTestUser(t, q)
	ctx := context.Background()

	token, _, err := auth.CreateSession(ctx, q, userID, "agent")
	if err != nil {
		t.Fatal(err)
	}
	first, err := auth.RefreshSession(ctx, q, token)
	if err != nil {
		t.Fatal(err)
	}

	replay, err := auth.RefreshSession(ctx, q, token)
	if err != nil {
		t.Fatalf("replay inside the grace period: %v", err)
	}
	if replay.Token != first.Token {
		t.Errorf("replay got %q, want the current token %q", replay.Token, first.Token)
	}
	if _, err := q.GetSession(ctx, first.Token); err != nil {
		t.Errorf("the current token stopped working after a replay: %v", err)
	}
}

// The replaced token coming back after the grace period means someone holds a
// copy. The session is ended, so neither holder keeps it.
func TestRefreshSessionRevokesOnReuseAfterTheGracePeriod(t *testing.T) {
	pool := testPool(t)
	q := sqlcgen.New(pool)
	userID := setupTestUser(t, q)
	ctx := context.Background()

	token, _, err := auth.CreateSession(ctx, q, userID, "agent")
	if err != nil {
		t.Fatal(err)
	}
	first, err := auth.RefreshSession(ctx, q, token)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "UPDATE sessions SET rotated_at = now() - $1::interval WHERE id = $2",
		(auth.RotationGrace + time.Second).String(), first.Token); err != nil {
		t.Fatal(err)
	}

	if _, err := auth.RefreshSession(ctx, q, token); !errors.Is(err, auth.ErrRefreshTokenReused) {
		t.Fatalf("reuse after the grace period = %v, want ErrRefreshTokenReused", err)
	}
	if _, err := auth.RefreshSession(ctx, q, first.Token); !errors.Is(err, auth.ErrInvalidRefreshToken) {
		t.Errorf("the owner's current token after a reuse = %v, want it revoked too", err)
	}
}

func TestRefreshSessionRejectsUnknownAndExpiredTokens(t *testing.T) {
	pool := testPool(t)
	q := sqlcgen.New(pool)
	userID := setupTestUser(t, q)
	ctx := context.Background()

	if _, err := auth.RefreshSession(ctx, q, "no-such-token"); !errors.Is(err, auth.ErrInvalidRefreshToken) {
		t.Errorf("unknown token = %v, want ErrInvalidRefreshToken", err)
	}

	token, _, err := auth.CreateSession(ctx, q, userID, "agent")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "UPDATE sessions SET expires_at = now() - interval '1 minute' WHERE id = $1", token); err != nil {
		t.Fatal(err)
	}
	if _, err := auth.RefreshSession(ctx, q, token); !errors.Is(err, auth.ErrInvalidRefreshToken) {
		t.Errorf("expired token = %v, want ErrInvalidRefreshToken", err)
	}
}
