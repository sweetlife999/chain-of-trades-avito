package token

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const testSecret = "test-secret-with-at-least-32-characters"

func TestManagerGeneratesAndParsesToken(t *testing.T) {
	t.Parallel()

	manager := NewManager(testSecret, 12*time.Hour)
	userID := uuid.New()

	raw, expiresAt, err := manager.Generate(userID)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if raw == "" {
		t.Fatal("Generate() returned an empty token")
	}
	if remaining := time.Until(expiresAt); remaining < 11*time.Hour+59*time.Minute || remaining > 12*time.Hour {
		t.Fatalf("token lifetime = %v, want about 12h", remaining)
	}

	actualID, err := manager.Parse(raw)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if actualID != userID {
		t.Fatalf("Parse() user id = %v, want %v", actualID, userID)
	}
}

func TestManagerRejectsInvalidTokens(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	tests := []struct {
		name  string
		build func(t *testing.T) string
	}{
		{
			name: "expired token",
			build: func(t *testing.T) string {
				raw, _, err := NewManager(testSecret, -time.Minute).Generate(userID)
				if err != nil {
					t.Fatalf("Generate() error = %v", err)
				}
				return raw
			},
		},
		{
			name: "wrong signature",
			build: func(t *testing.T) string {
				raw, _, err := NewManager("another-secret-with-at-least-32-chars", time.Hour).Generate(userID)
				if err != nil {
					t.Fatalf("Generate() error = %v", err)
				}
				return raw
			},
		},
		{
			name: "wrong algorithm",
			build: func(t *testing.T) string {
				claims := jwt.RegisteredClaims{
					Issuer:    issuer,
					Subject:   userID.String(),
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
				}
				raw, err := jwt.NewWithClaims(jwt.SigningMethodHS384, claims).SignedString([]byte(testSecret))
				if err != nil {
					t.Fatalf("SignedString() error = %v", err)
				}
				return raw
			},
		},
		{
			name: "invalid subject",
			build: func(t *testing.T) string {
				claims := jwt.RegisteredClaims{
					Issuer:    issuer,
					Subject:   "not-a-uuid",
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
				}
				raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
				if err != nil {
					t.Fatalf("SignedString() error = %v", err)
				}
				return raw
			},
		},
	}

	manager := NewManager(testSecret, time.Hour)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := manager.Parse(test.build(t))
			if !errors.Is(err, ErrInvalidToken) {
				t.Fatalf("Parse() error = %v, want ErrInvalidToken", err)
			}
		})
	}
}
