package auth

import (
	"context"
	"errors"
	"time"

	"expensetracker/internal/sqlcgen"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// verificationTokenTTL is longer than resetTokenTTL: proving an address is
// reachable is not urgent the way recovering a locked-out account is, and a
// verification link may sit unread in an inbox for a day before anyone
// gets to it.
const verificationTokenTTL = 24 * time.Hour

// One-time email-verification tokens, on the same lifecycle as the reset
// tokens next door (generateToken, expiry check, delete-on-consume) against a
// table of their own, so a leaked verification link can never be replayed as
// a session or reset token.

// CreateVerificationToken invalidates any verification tokens already
// issued to userID -- so an earlier link (from a resend, or a superseded
// email change) stops working the moment a new one is requested -- and
// issues a fresh one proving email.
func CreateVerificationToken(ctx context.Context, q *sqlcgen.Queries, userID int64, email string) (string, time.Time, error) {
	if err := q.DeleteEmailVerificationTokensForUser(ctx, userID); err != nil {
		return "", time.Time{}, err
	}

	token, err := generateToken()
	if err != nil {
		return "", time.Time{}, err
	}
	expiresAt := time.Now().Add(verificationTokenTTL)

	row, err := q.CreateEmailVerificationToken(ctx, sqlcgen.CreateEmailVerificationTokenParams{
		Token:     token,
		UserID:    userID,
		Email:     email,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return "", time.Time{}, err
	}
	return row.Token, row.ExpiresAt.Time, nil
}

// ValidateVerificationToken reports which user and email token belongs to,
// without consuming it.
func ValidateVerificationToken(ctx context.Context, q *sqlcgen.Queries, token string) (userID int64, email string, err error) {
	row, err := q.GetEmailVerificationToken(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, "", errors.New("verification token not found")
		}
		return 0, "", err
	}
	if time.Now().After(row.ExpiresAt.Time) {
		_ = q.DeleteEmailVerificationToken(ctx, token)
		return 0, "", errors.New("verification token expired")
	}
	return row.UserID, row.Email, nil
}

// ConsumeVerificationToken deletes token so it cannot be replayed once the
// address it proved has already been applied.
func ConsumeVerificationToken(ctx context.Context, q *sqlcgen.Queries, token string) error {
	return q.DeleteEmailVerificationToken(ctx, token)
}
