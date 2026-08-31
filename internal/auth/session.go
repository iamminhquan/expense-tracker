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

// The session lifecycle -- issue, validate, delete -- against the sessions
// table. Each function takes the query executor rather than hanging off a
// type holding one, so a caller inside a transaction can pass its own
// Queries.WithTx(tx); the reset- and verification-token files below follow
// the same shape against their own tables, which is what keeps a leaked link
// of one kind from ever being replayable as a token of another.

// CreateSession issues a new session for userID and returns the token and expiry.
func CreateSession(ctx context.Context, q *sqlcgen.Queries, userID int64, userAgent string) (string, time.Time, error) {
	token, err := generateToken()
	if err != nil {
		return "", time.Time{}, err
	}
	expiresAt := time.Now().Add(sessionTTL)

	session, err := q.CreateSession(ctx, sqlcgen.CreateSessionParams{
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
func ValidateSession(ctx context.Context, q *sqlcgen.Queries, token string) (int64, error) {
	session, err := q.GetSession(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, errors.New("session not found")
		}
		return 0, err
	}
	if time.Now().After(session.ExpiresAt.Time) {
		_ = q.DeleteSession(ctx, token)
		return 0, errors.New("session expired")
	}
	return session.UserID, nil
}

// DeleteSession removes token from the sessions table.
func DeleteSession(ctx context.Context, q *sqlcgen.Queries, token string) error {
	return q.DeleteSession(ctx, token)
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
