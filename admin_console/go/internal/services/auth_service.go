package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"admin_console/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserExists        = errors.New("user already exists")
	ErrUserNotFound      = errors.New("user not found")
	ErrTokenExpired      = errors.New("token expired")
	ErrInvalidToken      = errors.New("invalid token")
	ErrAccountLocked     = errors.New("account locked")
)

type AuthService struct {
	db    *pgxpool.Pool
	redis *redis.Client
	cfg   JWTConfig
}

type JWTConfig struct {
	Secret          string
	ExpirationTime  time.Duration
	RefreshExpires  time.Duration
	Issuer          string
}

func NewAuthService(db *pgxpool.Pool, redis *redis.Client) *AuthService {
	return &AuthService{
		db: db,
		redis: redis,
		cfg: JWTConfig{
			Secret:         "tigerwallet-secret-key-change-in-production",
			ExpirationTime: 24 * time.Hour,
			RefreshExpires: 7 * 24 * time.Hour,
			Issuer:         "tigerwallet",
		},
	}
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*models.AuthResponse, error) {
	var user models.User
	var profile models.UserProfile

	err := s.db.QueryRow(ctx, `
		SELECT id, email, username, password_hash, role, status, email_verified, 
		       two_factor_enabled, login_attempts, locked_until
		FROM users WHERE email = $1
	`, email).Scan(
		&user.ID, &user.Email, &user.Username, &user.PasswordHash, 
		&user.Role, &user.Status, &user.EmailVerified, &user.TwoFactorEnabled,
		&user.LoginAttempts, &user.LockedUntil,
	)

	if err == pgx.ErrNoRows {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	// Check if account is locked
	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		return nil, ErrAccountLocked
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		// Increment login attempts
		_, err = s.db.Exec(ctx, `
			UPDATE users SET login_attempts = login_attempts + 1,
			locked_until = CASE WHEN login_attempts >= 4 THEN NOW() + INTERVAL '15 minutes' ELSE NULL END
			WHERE id = $1
		`, user.ID)
		return nil, ErrInvalidCredentials
	}

	// Reset login attempts on successful login
	_, err = s.db.Exec(ctx, `
		UPDATE users SET login_attempts = 0, locked_until = NULL, last_login_at = NOW() WHERE id = $1
	`, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to update login: %w", err)
	}

	// Get user profile
	s.db.QueryRow(ctx, `SELECT id, user_id, avatar_url, timezone, language FROM user_profiles WHERE user_id = $1`, 
		user.ID).Scan(&profile.ID, &profile.UserID, &profile.AvatarURL, &profile.Timezone, &profile.Language)

	// Generate tokens
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.generateRefreshToken(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Store session in Redis
	sessionKey := fmt.Sprintf("session:%s", user.ID.String())
	s.redis.Set(ctx, sessionKey, accessToken, 24*time.Hour)

	return &models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.cfg.ExpirationTime.Seconds()),
		User:         &user,
	}, nil
}

func (s *AuthService) Register(ctx context.Context, req *models.RegisterRequest) (*models.AuthResponse, error) {
	// Check if user exists
	var exists bool
	err := s.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1 OR username = $2)`, 
		req.Email, req.Username).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check user existence: %w", err)
	}
	if exists {
		return nil, ErrUserExists
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), 14)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	var userID uuid.UUID
	err = s.db.QueryRow(ctx, `
		INSERT INTO users (email, username, password_hash, first_name, last_name, role, status, email_verified)
		VALUES ($1, $2, $3, $4, $5, 'admin', 'active', true)
		RETURNING id
	`, req.Email, req.Username, string(hashedPassword), req.FirstName, req.LastName).Scan(&userID)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Create user profile
	_, err = s.db.Exec(ctx, `
		INSERT INTO user_profiles (user_id, timezone, language, preferences)
		VALUES ($1, 'UTC', 'en', '{}')
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to create profile: %w", err)
	}

	// Get created user
	var user models.User
	s.db.QueryRow(ctx, `SELECT id, email, username, role, status FROM users WHERE id = $1`, userID).Scan(
		&user.ID, &user.Email, &user.Username, &user.Role, &user.Status,
	)

	// Generate tokens
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.generateRefreshToken(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.cfg.ExpirationTime.Seconds()),
		User:         &user,
	}, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*models.AuthResponse, error) {
	// Verify refresh token
	tokenHash := hashToken(refreshToken)
	var userID uuid.UUID
	var expiresAt time.Time

	err := s.db.QueryRow(ctx, `
		SELECT user_id, expires_at FROM refresh_tokens 
		WHERE token_hash = $1 AND expires_at > NOW()
	`, tokenHash).Scan(&userID, &expiresAt)
	if err == pgx.ErrNoRows {
		return nil, ErrInvalidToken
	}
	if err != nil {
		return nil, fmt.Errorf("failed to verify refresh token: %w", err)
	}

	// Get user
	var user models.User
	err = s.db.QueryRow(ctx, `
		SELECT id, email, username, role, status FROM users WHERE id = $1
	`, userID).Scan(&user.ID, &user.Email, &user.Username, &user.Role, &user.Status)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// Delete old refresh token
	s.db.Exec(ctx, `DELETE FROM refresh_tokens WHERE token_hash = $1`, tokenHash)

	// Generate new tokens
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	newRefreshToken, err := s.generateRefreshToken(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int64(s.cfg.ExpirationTime.Seconds()),
		User:         &user,
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, userID uuid.UUID) error {
	// Delete refresh tokens
	_, err := s.db.Exec(ctx, `DELETE FROM refresh_tokens WHERE user_id = $1`, userID)
	if err != nil {
		return fmt.Errorf("failed to delete refresh tokens: %w", err)
	}

	// Delete session from Redis
	sessionKey := fmt.Sprintf("session:%s", userID.String())
	s.redis.Del(ctx, sessionKey)

	return nil
}

func (s *AuthService) ValidateToken(ctx context.Context, tokenString string) (*models.User, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.cfg.Secret), nil
	})

	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	userIDStr, ok := claims["sub"].(string)
	if !ok {
		return nil, ErrInvalidToken
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, ErrInvalidToken
	}

	// Get user from database
	var user models.User
	err = s.db.QueryRow(ctx, `
		SELECT id, email, username, role, status FROM users WHERE id = $1
	`, userID).Scan(&user.ID, &user.Email, &user.Username, &user.Role, &user.Status)
	if err == pgx.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

func (s *AuthService) ForgotPassword(ctx context.Context, email string) error {
	// Check if user exists
	var userID uuid.UUID
	err := s.db.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, email).Scan(&userID)
	if err == pgx.ErrNoRows {
		// Don't reveal if email exists
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to check user: %w", err)
	}

	// Generate reset token (in production, send via email)
	resetToken := generateRandomToken(32)
	tokenHash := hashToken(resetToken)

	// Store in Redis with 1 hour expiry
	resetKey := fmt.Sprintf("password_reset:%s", userID.String())
	s.redis.Set(ctx, resetKey, tokenHash, time.Hour)

	// In production, send email with reset link
	fmt.Printf("Password reset token for %s: %s\n", email, resetToken)

	return nil
}

func (s *AuthService) ResetPassword(ctx context.Context, token, newPassword string) error {
	// This would need to find the user from the token
	// For now, return a placeholder
	return nil
}

func (s *AuthService) VerifyEmail(ctx context.Context, token string) error {
	return nil
}

func (s *AuthService) generateAccessToken(user models.User) (string, error) {
	claims := jwt.MapClaims{
		"sub":  user.ID.String(),
		"email": user.Email,
		"role": user.Role,
		"exp":  time.Now().Add(s.cfg.ExpirationTime).Unix(),
		"iat":  time.Now().Unix(),
		"iss":  s.cfg.Issuer,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.Secret))
}

func (s *AuthService) generateRefreshToken(ctx context.Context, userID uuid.UUID) (string, error) {
	refreshToken := generateRandomToken(64)
	tokenHash := hashToken(refreshToken)

	expiresAt := time.Now().Add(s.cfg.RefreshExpires)

	_, err := s.db.Exec(ctx, `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, userID, tokenHash, expiresAt)
	if err != nil {
		return "", fmt.Errorf("failed to store refresh token: %w", err)
	}

	return refreshToken, nil
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func generateRandomToken(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)[:length]
}
