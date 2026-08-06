// AuthService - Authentication and authorization service
package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tigerwallet/admin_panel/internal/config"
	"github.com/tigerwallet/admin_panel/internal/database"
	"github.com/tigerwallet/admin_panel/internal/middleware"
	"github.com/tigerwallet/admin_panel/internal/models"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserExists         = errors.New("user already exists")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token expired")
)

type AuthService struct {
	cfg *config.Config
}

func NewAuthService(cfg *config.Config) *AuthService {
	return &AuthService{cfg: cfg}
}

// Register creates a new admin user
func (s *AuthService) Register(ctx context.Context, username, email, password, role string) (*models.AdminUser, error) {
	// Check if user exists
	var existingID uuid.UUID
	err := database.QueryRow(ctx, 
		"SELECT id FROM admin_users WHERE email = $1 OR username = $2", 
		email, username,
	).Scan(&existingID)

	if err == nil {
		return nil, ErrUserExists
	}
	if err != pgx.ErrNoRows {
		return nil, err
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), s.cfg.BCryptCost)
	if err != nil {
		return nil, err
	}

	// Create user
	var admin models.AdminUser
	err = database.QueryRow(ctx, `
		INSERT INTO admin_users (username, email, password_hash, role, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, true, NOW(), NOW())
		RETURNING id, username, email, role, two_factor_enabled, is_active, created_at, updated_at
	`, username, email, string(hashedPassword), role).Scan(
		&admin.ID, &admin.Username, &admin.Email, &admin.Role,
		&admin.TwoFactorEnabled, &admin.IsActive, &admin.CreatedAt, &admin.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &admin, nil
}

// Login authenticates an admin user and returns JWT tokens
func (s *AuthService) Login(ctx context.Context, email, password, ipAddress, userAgent string) (*models.AdminUser, string, string, error) {
	var admin models.AdminUser
	var passwordHash string

	err := database.QueryRow(ctx, `
		SELECT id, username, email, password_hash, role, two_factor_enabled, is_active, created_at, updated_at, last_login
		FROM admin_users WHERE email = $1
	`, email).Scan(
		&admin.ID, &admin.Username, &admin.Email, &passwordHash, &admin.Role,
		&admin.TwoFactorEnabled, &admin.IsActive, &admin.CreatedAt, &admin.UpdatedAt, &admin.LastLogin,
	)

	if err == pgx.ErrNoRows {
		return nil, "", "", ErrInvalidCredentials
	}
	if err != nil {
		return nil, "", "", err
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		return nil, "", "", ErrInvalidCredentials
	}

	// Check if active
	if !admin.IsActive {
		return nil, "", "", errors.New("account is disabled")
	}

	// Generate tokens
	accessToken, err := s.generateAccessToken(&admin)
	if err != nil {
		return nil, "", "", err
	}

	refreshToken, err := s.generateRefreshToken(&admin)
	if err != nil {
		return nil, "", "", err
	}

	// Create session
	sessionToken := generateRandomString(32)
	sessionHash := hashToken(sessionToken)
	_, err = database.Exec(ctx, `
		INSERT INTO admin_sessions (admin_id, token_hash, ip_address, user_agent, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
	`, admin.ID, sessionHash, ipAddress, userAgent, time.Now().Add(s.cfg.JWTRefreshExpiry))

	if err != nil {
		return nil, "", "", err
	}

	// Update last login
	database.Exec(ctx, "UPDATE admin_users SET last_login = NOW() WHERE id = $1", admin.ID)

	return &admin, accessToken, refreshToken, nil
}

// LoginWith2FA authenticates with 2FA code
func (s *AuthService) LoginWith2FA(ctx context.Context, email, password, code, ipAddress, userAgent string) (*models.AdminUser, string, string, error) {
	admin, accessToken, refreshToken, err := s.Login(ctx, email, password, ipAddress, userAgent)
	if err != nil {
		return nil, "", "", err
	}

	// Verify 2FA code (simplified - in production use proper TOTP)
	// This is a placeholder for actual TOTP verification
	_ = code // Would verify against stored secret

	return admin, accessToken, refreshToken, nil
}

// RefreshToken refreshes an access token
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (string, error) {
	claims := &middleware.Claims{}
	token, err := jwtParse(refreshToken, s.cfg.JWTSecret, claims)
	if err != nil {
		return "", ErrInvalidToken
	}

	if !token.Valid {
		return "", ErrInvalidToken
	}

	// Get admin user
	var admin models.AdminUser
	err = database.QueryRow(ctx, `
		SELECT id, username, email, role, is_active FROM admin_users WHERE id = $1
	`, claims.AdminID).Scan(&admin.ID, &admin.Username, &admin.Email, &admin.Role, &admin.IsActive)

	if err != nil {
		return "", ErrUserNotFound
	}

	if !admin.IsActive {
		return "", errors.New("account is disabled")
	}

	return s.generateAccessToken(&admin)
}

// Logout invalidates a session
func (s *AuthService) Logout(ctx context.Context, adminID uuid.UUID, tokenHash string) error {
	_, err := database.Exec(ctx, `
		DELETE FROM admin_sessions WHERE admin_id = $1 AND token_hash = $2
	`, adminID, tokenHash)
	return err
}

// GetAdminByID retrieves an admin by ID
func (s *AuthService) GetAdminByID(ctx context.Context, id uuid.UUID) (*models.AdminUser, error) {
	var admin models.AdminUser
	err := database.QueryRow(ctx, `
		SELECT id, username, email, role, two_factor_enabled, is_active, created_at, updated_at, last_login
		FROM admin_users WHERE id = $1
	`, id).Scan(
		&admin.ID, &admin.Username, &admin.Email, &admin.Role,
		&admin.TwoFactorEnabled, &admin.IsActive, &admin.CreatedAt, &admin.UpdatedAt, &admin.LastLogin,
	)

	if err == pgx.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	return &admin, nil
}

// ListAdmins retrieves all admin users
func (s *AuthService) ListAdmins(ctx context.Context) ([]models.AdminUser, error) {
	rows, err := database.Query(ctx, `
		SELECT id, username, email, role, two_factor_enabled, is_active, created_at, updated_at, last_login
		FROM admin_users ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var admins []models.AdminUser
	for rows.Next() {
		var admin models.AdminUser
		err := rows.Scan(
			&admin.ID, &admin.Username, &admin.Email, &admin.Role,
			&admin.TwoFactorEnabled, &admin.IsActive, &admin.CreatedAt, &admin.UpdatedAt, &admin.LastLogin,
		)
		if err != nil {
			return nil, err
		}
		admins = append(admins, admin)
	}

	return admins, nil
}

// UpdateAdmin updates an admin user
func (s *AuthService) UpdateAdmin(ctx context.Context, id uuid.UUID, username, email, role string) error {
	_, err := database.Exec(ctx, `
		UPDATE admin_users SET username = $1, email = $2, role = $3, updated_at = NOW()
		WHERE id = $4
	`, username, email, role, id)
	return err
}

// DeleteAdmin deletes an admin user
func (s *AuthService) DeleteAdmin(ctx context.Context, id uuid.UUID) error {
	_, err := database.Exec(ctx, "DELETE FROM admin_users WHERE id = $1", id)
	return err
}

// ChangePassword changes admin password
func (s *AuthService) ChangePassword(ctx context.Context, adminID uuid.UUID, oldPassword, newPassword string) error {
	var passwordHash string
	err := database.QueryRow(ctx, "SELECT password_hash FROM admin_users WHERE id = $1", adminID).Scan(&passwordHash)
	if err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(oldPassword)); err != nil {
		return errors.New("invalid old password")
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), s.cfg.BCryptCost)
	if err != nil {
		return err
	}

	_, err = database.Exec(ctx, "UPDATE admin_users SET password_hash = $1, updated_at = NOW() WHERE id = $2", string(newHash), adminID)
	return err
}

// Enable2FA enables 2FA for an admin
func (s *AuthService) Enable2FA(ctx context.Context, adminID uuid.UUID, secret string) error {
	_, err := database.Exec(ctx, `
		UPDATE admin_users SET two_factor_secret = $1, two_factor_enabled = true, updated_at = NOW()
		WHERE id = $2
	`, secret, adminID)
	return err
}

// Disable2FA disables 2FA for an admin
func (s *AuthService) Disable2FA(ctx context.Context, adminID uuid.UUID) error {
	_, err := database.Exec(ctx, `
		UPDATE admin_users SET two_factor_secret = NULL, two_factor_enabled = false, updated_at = NOW()
		WHERE id = $1
	`, adminID)
	return err
}

// SuspendAdmin suspends an admin user
func (s *AuthService) SuspendAdmin(ctx context.Context, id uuid.UUID) error {
	_, err := database.Exec(ctx, "UPDATE admin_users SET is_active = false WHERE id = $1", id)
	return err
}

// ActivateAdmin activates an admin user
func (s *AuthService) ActivateAdmin(ctx context.Context, id uuid.UUID) error {
	_, err := database.Exec(ctx, "UPDATE admin_users SET is_active = true WHERE id = $1", id)
	return err
}

func (s *AuthService) generateAccessToken(admin *models.AdminUser) (string, error) {
	claims := middleware.Claims{
		AdminID:  admin.ID,
		Username: admin.Username,
		Email:    admin.Email,
		Role:     admin.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.cfg.JWTExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwtSign(claims, s.cfg.JWTSecret)
}

func (s *AuthService) generateRefreshToken(admin *models.AdminUser) (string, error) {
	claims := middleware.Claims{
		AdminID:  admin.ID,
		Username: admin.Username,
		Email:    admin.Email,
		Role:     admin.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.cfg.JWTRefreshExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwtSign(claims, s.cfg.JWTSecret)
}

func generateRandomString(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:length]
}

func hashToken(token string) string {
	hash, _ := bcrypt.GenerateFromPassword([]byte(token), 10)
	return string(hash)
}

// JWT helper functions
func jwtSign(claims middleware.Claims, secret string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func jwtParse(tokenString string, secret string, claims *middleware.Claims) (*jwt.Token, error) {
	return jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
}
