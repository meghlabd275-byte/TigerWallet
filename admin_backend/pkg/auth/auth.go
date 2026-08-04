package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"admin_backend/internal/config"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// AuthService handles authentication operations
type AuthService struct {
	cfg *config.Config
}

// NewAuthService creates a new auth service
func NewAuthService(cfg *config.Config) *AuthService {
	return &AuthService{cfg: cfg}
}

// TokenClaims represents JWT claims
type TokenClaims struct {
	AdminID uint   `json:"admin_id"`
	Email   string `json:"email"`
	Role    string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken generates a JWT token
func (s *AuthService) GenerateToken(adminID uint, email, role string, expiresAt time.Time) (string, error) {
	claims := TokenClaims{
		AdminID: adminID,
		Email:   email,
		Role:    role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "tigerwallet-admin",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWTSecret))
}

// GenerateRefreshToken generates a refresh token
func (s *AuthService) GenerateRefreshToken(adminID uint) (string, error) {
	// Generate a random token
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	token := base64.URLEncoding.EncodeToString(bytes)

	// Add prefix and admin ID for identification
	return fmt.Sprintf("rt_%d_%s", adminID, token), nil
}

// VerifyToken verifies a JWT token
func (s *AuthService) VerifyToken(tokenString string) (map[string]interface{}, error) {
	// Remove "Bearer " prefix if present
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")

	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.cfg.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*TokenClaims); ok && token.Valid {
		return map[string]interface{}{
			"admin_id": claims.AdminID,
			"email":    claims.Email,
			"role":     claims.Role,
			"exp":      claims.ExpiresAt,
			"iat":      claims.IssuedAt,
		}, nil
	}

	return nil, errors.New("invalid token")
}

// HashPassword hashes a password with pepper
func (s *AuthService) HashPassword(password string) (string, error) {
	hashedPassword := password + s.cfg.PasswordPepper
	bytes, err := bcrypt.GenerateFromPassword([]byte(hashedPassword), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// VerifyPassword verifies a password against a hash
func (s *AuthService) VerifyPassword(password, hash string) bool {
	hashedPassword := password + s.cfg.PasswordPepper
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(hashedPassword))
	return err == nil
}

// GenerateAPIKey generates an API key
func (s *AuthService) GenerateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// HashAPIKey hashes an API key for storage
func (s *AuthService) HashAPIKey(apiKey string) string {
	bytes := []byte(apiKey)
	hash := make([]byte, base64.StdEncoding.EncodedLen(len(bytes)))
	base64.StdEncoding.Encode(hash, bytes)
	return string(hash)
}

// ValidatePassword validates password strength
func (s *AuthService) ValidatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}

	hasUpper := false
	hasLower := false
	hasDigit := false
	hasSpecial := false

	for _, char := range password {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasDigit = true
		case strings.ContainsRune("!@#$%^&*()_+-=[]{}|;':\",./<>?", char):
			hasSpecial = true
		}
	}

	if !hasUpper {
		return errors.New("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return errors.New("password must contain at least one lowercase letter")
	}
	if !hasDigit {
		return errors.New("password must contain at least one digit")
	}
	if !hasSpecial {
		return errors.New("password must contain at least one special character")
	}

	return nil
}
