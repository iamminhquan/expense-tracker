package auth_test

import (
	"testing"

	"expensetracker/internal/auth"
)

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := auth.HashPassword("correct-horse")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if hash == "correct-horse" {
		t.Fatal("hash must not equal plaintext")
	}
	if !auth.VerifyPassword(hash, "correct-horse") {
		t.Fatal("expected verify to succeed for correct password")
	}
	if auth.VerifyPassword(hash, "wrong-password") {
		t.Fatal("expected verify to fail for wrong password")
	}
}
