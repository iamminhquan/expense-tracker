package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var ErrInvalidToken = errors.New("invalid token")

type Claims struct {
	Subject   string `json:"sub"`
	ExpiresAt int64  `json:"exp"`
}

func GenerateToken(userID uint, secret string, ttl time.Duration) (string, error) {
	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}
	claims := Claims{
		Subject:   strconv.FormatUint(uint64(userID), 10),
		ExpiresAt: time.Now().Add(ttl).Unix(),
	}

	headerPart, err := encodeJSON(header)
	if err != nil {
		return "", err
	}

	payloadPart, err := encodeJSON(claims)
	if err != nil {
		return "", err
	}

	signingInput := headerPart + "." + payloadPart
	signature := sign(signingInput, secret)

	return signingInput + "." + signature, nil
}

func ParseToken(token string, secret string) (uint, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return 0, ErrInvalidToken
	}

	signingInput := parts[0] + "." + parts[1]
	expectedSignature := sign(signingInput, secret)
	if !hmac.Equal([]byte(parts[2]), []byte(expectedSignature)) {
		return 0, ErrInvalidToken
	}

	var claims Claims
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return 0, ErrInvalidToken
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return 0, ErrInvalidToken
	}
	if claims.ExpiresAt < time.Now().Unix() {
		return 0, ErrInvalidToken
	}

	parsedID, err := strconv.ParseUint(claims.Subject, 10, 64)
	if err != nil {
		return 0, ErrInvalidToken
	}

	return uint(parsedID), nil
}

func encodeJSON(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("marshal token data: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(data), nil
}

func sign(input string, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(input))

	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
