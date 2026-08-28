package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"expensetracker/internal/sqlcgen"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const sessionTTL = 7 * 24 * time.Hour

// Manager issues and validates session tokens against the sessions table.
type Manager struct {
	queries *sqlcgen.Queries
}

// NewManager constructs a session Manager backed by the given query executor.
func NewManager(q *sqlcgen.Queries) *Manager {
	return &Manager{queries: q}
}

// CreateSession issues a new session for userID and returns the token and expiry.
func (m *Manager) CreateSession(ctx context.Context, userID int64, userAgent string) (string, time.Time, error) {
	token, err := generateToken()
	if err != nil {
		return "", time.Time{}, err
	}
	expiresAt := time.Now().Add(sessionTTL)

	session, err := m.queries.CreateSession(ctx, sqlcgen.CreateSessionParams{
		ID:        token,
		UserID:    userID,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
		UserAgent: pgtype.Text{String: userAgent, Valid: userAgent != ""},
	})
	if err != nil {
		return "", time.Time{}, err
	}
	return session.ID, session.ExpiresAt.Time, nil
}

// ValidateSession reports which user token belongs to, or an error if the
// session is not found or has expired.
func (m *Manager) ValidateSession(ctx context.Context, token string) (int64, error) {
	session, err := m.queries.GetSession(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, errors.New("session not found")
		}
		return 0, err
	}
	if time.Now().After(session.ExpiresAt.Time) {
		_ = m.queries.DeleteSession(ctx, token)
		return 0, errors.New("session expired")
	}
	return session.UserID, nil
}

// DeleteSession removes token from the sessions table.
func (m *Manager) DeleteSession(ctx context.Context, token string) error {
	return m.queries.DeleteSession(ctx, token)
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
