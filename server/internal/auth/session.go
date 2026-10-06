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

// ErrInvalidRefreshToken means the token is unknown, expired, or was replaced
// long enough ago that it can no longer be told apart from one never issued.
var ErrInvalidRefreshToken = errors.New("invalid refresh token")

// ErrRefreshTokenReused means a token that had already been replaced came back
// after RotationGrace. The legitimate client holds the newer token by then, so
// this is a copy being replayed, and the session it belonged to has been ended.
var ErrRefreshTokenReused = errors.New("refresh token reused")

// RotationGrace is how long a replaced token still works. Two requests can
// carry one token honestly: a second browser tab refreshing at the same moment,
// a retry after the response carrying the new token was lost. Past it, the
// replaced token is evidence of a stolen copy.
const RotationGrace = 30 * time.Second

// RefreshedSession is what a client holds after a refresh.
type RefreshedSession struct {
	UserID int64
	// Token is the refresh token the client should now send. It is the one
	// just minted, or, for a request inside the grace period, the one the
	// racing request was already given.
	Token     string
	ExpiresAt time.Time
}

// RefreshSession exchanges a refresh token for its successor. The session's
// expiry does not move: rotating a token limits how long a stolen copy is
// useful, it does not extend how long the login lasts.
//
// A token already replaced is not rotated again. Inside RotationGrace it gets
// the current token back; after, the session is deleted and
// ErrRefreshTokenReused returned, so a thief and the owner replaying the same
// token both lose it rather than the thief keeping it quietly.
func RefreshSession(ctx context.Context, q *sqlcgen.Queries, token string) (RefreshedSession, error) {
	next, err := generateToken()
	if err != nil {
		return RefreshedSession{}, err
	}

	rotated, err := q.RotateSession(ctx, sqlcgen.RotateSessionParams{ID: token, ID_2: next})
	if err == nil {
		return RefreshedSession{UserID: rotated.UserID, Token: rotated.ID, ExpiresAt: rotated.ExpiresAt.Time}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return RefreshedSession{}, err
	}

	// Unknown, expired, or already replaced: only the last is worth a second look.
	replaced, err := q.GetSessionByPreviousID(ctx, pgtype.Text{String: token, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RefreshedSession{}, ErrInvalidRefreshToken
		}
		return RefreshedSession{}, err
	}
	if time.Now().After(replaced.ExpiresAt.Time) {
		return RefreshedSession{}, ErrInvalidRefreshToken
	}
	if time.Since(replaced.RotatedAt.Time) > RotationGrace {
		if err := q.DeleteSession(ctx, replaced.ID); err != nil {
			return RefreshedSession{}, err
		}
		return RefreshedSession{}, ErrRefreshTokenReused
	}
	return RefreshedSession{UserID: replaced.UserID, Token: replaced.ID, ExpiresAt: replaced.ExpiresAt.Time}, nil
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
