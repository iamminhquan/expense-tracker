package auth

import (
	"context"
	"errors"
	"time"

	"expensetracker/internal/sqlcgen"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// resetTokenTTL is much shorter than sessionTTL: a reset link sits in an
// inbox and might be forwarded or left unread, so it should not stay valid
// for days the way a signed-in session does.
const resetTokenTTL = 1 * time.Hour

// One-time password-reset tokens. They mirror the session lifecycle in
// session.go (generateToken, expiry check, delete-on-consume) against a table
// of their own, so a leaked reset link can never be replayed as a session
// token or vice versa.

// CreateResetToken invalidates any reset tokens already issued to userID --
// so an old, possibly-leaked link stops working the moment a new one is
// requested -- and issues a fresh one.
func CreateResetToken(ctx context.Context, q *sqlcgen.Queries, userID int64) (string, time.Time, error) {
	if err := q.DeletePasswordResetTokensForUser(ctx, userID); err != nil {
		return "", time.Time{}, err
	}

	token, err := generateToken()
	if err != nil {
		return "", time.Time{}, err
	}
	expiresAt := time.Now().Add(resetTokenTTL)

	row, err := q.CreatePasswordResetToken(ctx, sqlcgen.CreatePasswordResetTokenParams{
		Token:     token,
		UserID:    userID,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return "", time.Time{}, err
	}
	return row.Token, row.ExpiresAt.Time, nil
}

// ValidateResetToken reports which user token belongs to, without consuming
// it. The reset-password page calls this on the initial GET so a dead link
// shows as invalid immediately, before the visitor has typed anything.
func ValidateResetToken(ctx context.Context, q *sqlcgen.Queries, token string) (int64, error) {
	row, err := q.GetPasswordResetToken(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, errors.New("reset token not found")
		}
		return 0, err
	}
	if time.Now().After(row.ExpiresAt.Time) {
		_ = q.DeletePasswordResetToken(ctx, token)
		return 0, errors.New("reset token expired")
	}
	return row.UserID, nil
}

// ConsumeResetToken deletes token so it cannot be replayed once the
// password it authorized has already been changed.
func ConsumeResetToken(ctx context.Context, q *sqlcgen.Queries, token string) error {
	return q.DeletePasswordResetToken(ctx, token)
}
